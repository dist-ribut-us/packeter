package packeter

import (
	"crypto/rand"
	"github.com/dist-ribut-us/rnet"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestUint32Marshal(t *testing.T) {
	ints := []uint32{0, 1, 31415}
	for _, i := range ints {
		b := make([]byte, 4)
		marshalUint32(i, b)
		assert.Equal(t, i, unmarshalUint32(b))
	}
}

func TestFindRedundancy(t *testing.T) {
	dataShards, parityShards := findRedundancy(20000, 1000, .05, .999)
	assert.Equal(t, 20, dataShards)
	assert.Equal(t, 6, parityShards)
}

func TestMake(t *testing.T) {
	ln := int(float32(Packetlength) * 4.5)
	loss := .05
	reliability := .999

	p := New()
	msg := make([]byte, ln)
	rand.Read(msg)
	packetBytes, err := p.Make(msg, loss, reliability)
	assert.NoError(t, err)
	assert.Equal(t, 8, len(packetBytes))

	pks := make([]*Packet, len(packetBytes))
	for i, b := range packetBytes {
		pks[i] = Unmarshal(b)
	}

	id := pks[0].MessageID
	ds := pks[0].DataShards()
	ps := pks[0].ParityShards
	pl := pks[0].Packets
	for i, pk := range pks {
		assert.Equal(t, uint16(i), pk.PacketID)
		assert.Equal(t, id, pk.MessageID)
		assert.Equal(t, ds, pk.DataShards())
		assert.Equal(t, ps, pk.ParityShards)
		assert.Equal(t, pl, pk.Packets)
	}
}

func TestRoundTrip(t *testing.T) {
	ln := int(float32(Packetlength) * 4.5)
	loss := .05
	reliability := .999

	p := New()
	msg := make([]byte, ln)
	rand.Read(msg)
	pks, err := p.Make(msg, loss, reliability)
	assert.NoError(t, err)
	addr, err := rnet.ResolveAddr("127.0.0.1:1234")
	assert.NoError(t, err)
	go func() {
		for i := 0; i < len(pks); i++ {
			if i == 1 {
				// skip sending packet #1
				// tests packet loss
				continue
			}
			p.Receive(pks[i], addr)
		}
	}()

	out := <-p.Chan()
	assert.NoError(t, out.Err)
	assert.Equal(t, msg, out.Body)
	assert.Equal(t, addr.String(), out.Addr.String())
}

func TestPacketMarshalUnmarshal(t *testing.T) {
	msg := make([]byte, 100)
	rand.Read(msg)

	pk1 := &Packet{
		MessageID:    1,
		PacketID:     2,
		ParityShards: 3,
		Packets:      4,
		Data:         msg,
	}

	pk2 := Unmarshal(pk1.Marshal())
	assert.Equal(t, pk1, pk2)
}

func TestNoPairityRequired(t *testing.T) {
	ln := int(float32(Packetlength) * 4.5)
	loss := .001
	reliability := .99

	p := New()
	msg := make([]byte, ln)
	rand.Read(msg)
	pks, err := p.Make(msg, loss, reliability)
	assert.NoError(t, err)
	addr, err := rnet.ResolveAddr("127.0.0.1:1234")
	assert.NoError(t, err)
	go func() {
		for i := 0; i < len(pks); i++ {
			if i == 1 {
				continue
			}
			p.Receive(pks[i], addr)
		}
	}()

	out := <-p.Chan()
	assert.NoError(t, out.Err)
	assert.Equal(t, msg, out.Body)
	assert.Equal(t, addr.String(), out.Addr.String())
}

func TestTTL(t *testing.T) {
	oldTTL := TTL
	TTL = time.Millisecond

	ln := int(float32(Packetlength) * 4.5)
	loss := .05
	reliability := .999

	p := New()
	msg := make([]byte, ln)
	rand.Read(msg)
	pks, err := p.Make(msg, loss, reliability)
	assert.NoError(t, err)
	addr, err := rnet.ResolveAddr("127.0.0.1:1234")
	assert.NoError(t, err)

	p.Receive(pks[0], addr)
	assert.Equal(t, 1, len(p.collectors))

	time.Sleep(time.Millisecond * 2)
	assert.Equal(t, 0, len(p.collectors))

	m := <-p.Chan()
	assert.Equal(t, errTimedOut.Error(), m.Err.Error())

	TTL = oldTTL
}
