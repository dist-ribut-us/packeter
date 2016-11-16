// Packeter breaks a message into Reed-Solomon shards with enough parity shards
// to accommodate for expected losses.
package packeter

import (
	"bytes"
	"errors"
	"github.com/dist-ribut-us/crypto"
	"github.com/dist-ribut-us/rnet"
	"github.com/ematvey/gostat"
	"github.com/klauspost/reedsolomon"
	"sync"
	"time"
)

// Packeter manages the process of collecting packets and resolving them to
// messages as well as converting a message into a set of packets
type Packeter struct {
	collectors   map[uint32]*collector
	packetLength int
	ch           chan *Message
	running      bool
	mtxRun       sync.Mutex
}

// start will start the loop the deletes timed out collectors if it is not
// running
func (p *Packeter) start() {
	p.mtxRun.Lock()
	if !p.running {
		go p.run()
	}
	p.mtxRun.Unlock()
}

// Chan returns the channel that can receive Messages after the packeter has
// reassembled them
func (p *Packeter) Chan() <-chan *Message {
	return p.ch
}

// Message represents a message that was transmitted.
type Message struct {
	ID   uint32
	Addr *rnet.Addr
	Body []byte
	Err  error
}

// New returns a new Packeter
func New() *Packeter {
	p := &Packeter{
		collectors:   make(map[uint32]*collector),
		packetLength: Packetlength,
		ch:           make(chan *Message, BufferSize),
	}
	return p
}

// TTL sets the time that a partial message will wait until is deleted
var TTL = time.Second * 10

var timedOut = errors.New("Timed Out")

// run will periodically clear out collectors that have timed out. When there
// are no collectors, the thread will exit.
func (p *Packeter) run() {
	p.mtxRun.Lock()
	p.running = true
	p.mtxRun.Unlock()
	keepRunning := true
	for keepRunning {
		time.Sleep(TTL)
		now := time.Now()
		var remove []uint32
		keepRunning = false
		for id, clctr := range p.collectors {
			if clctr.ttl.Before(now) {
				remove = append(remove, id)
				p.ch <- &Message{
					ID:   id,
					Addr: clctr.addr,
					Err:  timedOut,
				}
			} else {
				keepRunning = true
			}
		}
		for _, id := range remove {
			delete(p.collectors, id)
		}
	}
	p.mtxRun.Lock()
	p.running = false
	p.mtxRun.Unlock()
}

// collector collects the individual packets for a message, when enough packets
// have arrived the message can be collected
type collector struct {
	data      [][]byte
	collected map[uint16]bool
	complete  bool
	addr      *rnet.Addr
	ttl       time.Time
}

// Packetlength is the max packet length in bytes. Defaults to 10k. This cannot
// be higher than 65535 because of the UDP standard.
var Packetlength = 10000

// BufferSize is the default channel buffer size for a Packeter
var BufferSize = 50

// findRedundancy calculates the number of data and parity shards needed for a
// given message length, loss rate and target reliability. Right now, the
// algorithm just increases the number of parity shards until reliability is
// above the target reliability. There's probably a better way to do this, but I
// haven't found it yet.
func findRedundancy(dataSize, maxSize int, loss, reliability float64) (int, int) {
	// there's a better way to do this, but I need to do more research
	// for now, this works
	dataShards := dataSize / maxSize
	if dataSize%maxSize != 0 {
		dataShards++
	}
	parityShards := 0
	for stat.Binomial_CDF_At(loss, int64(dataShards+parityShards), int64(parityShards)) < reliability {
		parityShards++
	}
	return dataShards, parityShards
}

// Make takes a message as a byte slice, along with the expected loss rate and
// target reliability and produces the packets for that message. The packets are
// returned as a slice of byte-slices.
func (p *Packeter) Make(msg []byte, loss, reliability float64) ([][]byte, error) {
	l := uint32(len(msg) + 4)
	lb := []byte{
		byte(l),
		byte(l >> 8),
		byte(l >> 16),
		byte(l >> 24),
	}
	msg = append(lb, msg...)

	dataShards, parityShards := findRedundancy(len(msg), p.packetLength, loss, reliability)
	if parityShards < 1 {
		// reedsolomon.Encoder cannot have 0 parity shards
		// at some point I want to change this so it doesn't use reedsolomon in this
		// case
		parityShards = 1
	}
	shards := dataShards + parityShards

	var data [][]byte
	enc, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		return nil, err
	}
	data, err = enc.Split(msg)
	if err != nil {
		return nil, err
	}
	if err := enc.Encode(data); err != nil {
		return nil, err
	}

	pk := Packet{
		MessageId:    crypto.RandUint32(),
		Packets:      uint16(shards),
		ParityShards: uint16(parityShards),
	}
	pks := make([][]byte, shards)

	pk.ParityShards = uint16(parityShards)
	for i := 0; i < shards; i++ {
		pk.PacketId = uint16(i)
		pk.Data = data[i]
		pks[i] = pk.Marshal()
	}
	return pks, nil
}

// Receive collects packets. When exactly enough packets have been recovered to
// reconstruct the message, the message is returned as a byte slice. Otherwise
// nil is returned. Receive can continue to collect packets after the message
// has been constructed for reliability statistics.
func (p *Packeter) Receive(b []byte, addr *rnet.Addr) {
	pk := Unmarshal(b)
	clctr, ok := p.collectors[pk.MessageId]
	if !ok {
		clctr = &collector{
			data:      make([][]byte, pk.Packets),
			collected: make(map[uint16]bool),
			complete:  false,
			addr:      addr,
		}
		p.collectors[pk.MessageId] = clctr
	} else if addr.String() != clctr.addr.String() {
		return
	}
	clctr.ttl = time.Now().Add(TTL)
	p.start()
	clctr.collected[pk.PacketId] = true
	if clctr.complete {
		return
	}
	clctr.data[pk.PacketId] = pk.Data
	dataShards := int(pk.DataShards())
	if len(clctr.collected) < dataShards {
		return
	}
	clctr.complete = true
	enc, err := reedsolomon.New(dataShards, int(pk.ParityShards))
	if err != nil {
		p.ch <- &Message{
			ID:   pk.MessageId,
			Err:  err,
			Addr: addr,
		}
		return
	}
	err = enc.Reconstruct(clctr.data)
	if err != nil {
		p.ch <- &Message{
			ID:   pk.MessageId,
			Err:  err,
			Addr: addr,
		}
		return
	}

	ln := int(unmarshalUint32(clctr.data[0]))
	var out bytes.Buffer
	err = enc.Join(&out, clctr.data, ln)
	clctr.data = nil
	p.ch <- &Message{
		ID:   pk.MessageId,
		Err:  err,
		Addr: addr,
		Body: out.Bytes()[4:],
	}

}
