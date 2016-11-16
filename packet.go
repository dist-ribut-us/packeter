package packeter

const overhead = 4 + 3*2

// Packet represents a shard of a message. If enough shards from a message are
// collected, the message can be reconstructed.
type Packet struct {
	MessageID    uint32
	PacketID     uint16
	ParityShards uint16
	Packets      uint16
	Data         []byte
}

// Marshal serializes a Packet to a byte slice
func (p *Packet) Marshal() []byte {
	b := make([]byte, overhead+len(p.Data))
	marshalUint32(p.MessageID, b)
	marshalUint16(p.PacketID, b[4:])
	marshalUint16(p.ParityShards, b[6:])
	marshalUint16(p.Packets, b[8:])
	copy(b[overhead:], p.Data)
	return b
}

// DataShards returns the number of DataShards in a message, this is also the
// number of shards needed to reconstruct the message
func (p *Packet) DataShards() uint16 {
	return p.Packets - p.ParityShards
}

// Unmarshal deserializes a byte slice to a Packet
func Unmarshal(b []byte) *Packet {
	return &Packet{
		MessageID:    unmarshalUint32(b),
		PacketID:     unmarshalUint16(b[4:]),
		ParityShards: unmarshalUint16(b[6:]),
		Packets:      unmarshalUint16(b[8:]),
		Data:         b[overhead:],
	}
}

func marshalUint32(ui uint32, b []byte) { marshalUint(uint64(ui), 4, b) }
func unmarshalUint32(b []byte) uint32   { return uint32(unmarshalUint(4, b)) }

func marshalUint16(ui uint16, b []byte) { marshalUint(uint64(ui), 2, b) }
func unmarshalUint16(b []byte) uint16   { return uint16(unmarshalUint(2, b)) }

func marshalUint(ui uint64, l int, b []byte) {
	for i := 0; i < l; i++ {
		b[i] = byte(ui)
		ui >>= 8
	}
}

func unmarshalUint(l int, b []byte) uint64 {
	var ui uint64
	for i := l - 1; i >= 0; i-- {
		ui <<= 8
		ui += uint64(b[i])
	}
	return ui
}
