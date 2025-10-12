// dftp/packet.go implements the packet format and methods used by the DFTP protocol.

package dftp

import (
	"fmt"
	"hash/crc32"
)

type (
	MessageType uint8
	Flags       uint8
)

type Packet struct {
	// Header
	Type      MessageType
	StreamID  uint8  // 1 byte
	Length    uint16 // 2 bytes
	SessionID uint32 // 4 bytes
	ChunkNum  uint32 // 4 bytes

	// Data
	Checksum uint32 // 4 bytes
	Data     []byte
}

// NewPacket creates a new empty packet.
// The packet is not valid until it is filled with data.
func NewEmptyPacket() *Packet {
	return &Packet{
		Type:      0,
		Length:    0,
		SessionID: 0,
		ChunkNum:  0,
		Checksum:  0,
		Data:      nil,
	}
}

// Deserialize deserializes a packet from a byte slice.
// Returns an error if the packet data is invalid.
func Deserialize(data []byte) (*Packet, error) {
	p := NewEmptyPacket()
	if len(data) < HEADER_SIZE {
		return nil, fmt.Errorf("deserialize: packet too short or invalid")
	}

	p.Type = MessageType(data[0])
	p.StreamID = data[1]
	p.Length = uint16(data[2])<<8 | uint16(data[3])
	p.SessionID = uint32(data[4])<<24 | uint32(data[5])<<16 | uint32(data[6])<<8 | uint32(data[7])
	p.ChunkNum = uint32(data[8])<<24 | uint32(data[9])<<16 | uint32(data[10])<<8 | uint32(data[11])
	p.Checksum = uint32(data[12])<<24 | uint32(data[13])<<16 | uint32(data[14])<<8 | uint32(data[15])
	p.Data = make([]byte, len(data[16:]))
	copy(p.Data, data[16:])

	//if err := p.Validate(); err != nil {
	//	return err, nil
	//}
	return p, nil
}

// Serialize serializes the packet into a byte slice.
// The packet must be valid before calling this method.
func (p *Packet) Serialize() []byte {
	if p.Length == 0 {
		p.Length = uint16(len(p.Data))
	}

	data := make([]byte, HEADER_SIZE+len(p.Data))
	data[0] = byte(p.Type)
	data[1] = byte(p.StreamID)
	data[2] = byte(p.Length >> 8)
	data[3] = byte(p.Length)
	data[4] = byte(p.SessionID >> 24)
	data[5] = byte(p.SessionID >> 16)
	data[6] = byte(p.SessionID >> 8)
	data[7] = byte(p.SessionID)
	data[8] = byte(p.ChunkNum >> 24)
	data[9] = byte(p.ChunkNum >> 16)
	data[10] = byte(p.ChunkNum >> 8)
	data[11] = byte(p.ChunkNum)
	data[12] = byte(p.Checksum >> 24)
	data[13] = byte(p.Checksum >> 16)
	data[14] = byte(p.Checksum >> 8)
	data[15] = byte(p.Checksum)
	copy(data[16:], p.Data)

	return data
}

// Validate validates the packet.
// TODO: implement MesageType validation
func (p *Packet) Validate() error {
	if p.Length == 0 && len(p.Data) != 0 {
		return fmt.Errorf("packet length is invalid")
	}
	if p.Checksum == 0 {
		return fmt.Errorf("packet checksum is zero")
	}
	return nil
}

// Validate validates the raw packet data.
func Validate(data []byte) error {
	packet, err := Deserialize(data)
	if err != nil {
		return err
	}
	if err := packet.Validate(); err != nil {
		return err
	}
	return nil
}

// CalcChecksum calculates the checksum of the packet.
// Makes sure to only calculate the checksum once.
func (p *Packet) CalcChecksum() uint32 {
	if p.Checksum != 0 {
		return p.Checksum
	}
	data := p.Serialize()
	return crc32.ChecksumIEEE(data)
}

func (p *Packet) String() string {
	return ""
}

func (conn *Connection) NewPacketFromConn(t MessageType, data []byte) *Packet {
	p := &Packet{
		Type:     t,
		StreamID: conn.ConnID,
		Length:   uint16(len(data)),
		Data:     data,
	}
	// Not yet
	// p.CalcChecksum()
	return p
}
