package packeter

import (
	"github.com/dist-ribut-us/serial"
)

// Packet represents a shard of a message. If enough shards from a message are
// collected, the message can be reconstructed.
type Packet struct {
	PackageID    uint32
	PacketID     uint16
	ParityShards uint16
	Packets      uint16
	Data         []byte
}

// PackageID is 4, 2 for the PacketID, ParityShards and Packets each.
const overhead = 4 + 3*2

// Marshal serializes a Packet to a byte slice. Prepend allows additional
// meta-data to be added to the begining of each packet.
func (p *Packet) Marshal(tag []byte) []byte {
	tl := len(tag)
	b := make([]byte, overhead+len(p.Data)+tl)
	copy(b, tag)
	serial.MarshalUint32(p.PackageID, b[tl:])
	serial.MarshalUint16(p.PacketID, b[4+tl:])
	serial.MarshalUint16(p.ParityShards, b[6+tl:])
	serial.MarshalUint16(p.Packets, b[8+tl:])
	copy(b[overhead+tl:], p.Data)
	return b
}

// DataShards returns the number of DataShards in a message, this is also the
// number of shards needed to reconstruct the message
func (p *Packet) DataShards() uint16 {
	return p.Packets - p.ParityShards
}

// Unmarshal deserializes a byte slice to a Packet
func Unmarshal(b []byte) *Packet {
	if len(b) < overhead {
		return nil
	}
	return &Packet{
		PackageID:    serial.UnmarshalUint32(b),
		PacketID:     serial.UnmarshalUint16(b[4:]),
		ParityShards: serial.UnmarshalUint16(b[6:]),
		Packets:      serial.UnmarshalUint16(b[8:]),
		Data:         b[overhead:],
	}
}
