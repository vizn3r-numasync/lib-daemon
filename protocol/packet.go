package protocol

import (
	"fmt"
	"hash/crc32"
)

type (
	MessageType byte
	Flags       byte
)

type Packet struct {
	// Header
	Type      MessageType
	Flags     Flags
	Length    uint16 // 2 bytes
	SessionID uint32 // 4 bytes
	SEQNum    uint32 // 4 bytes
	ACKNum    uint32 // 4 bytes
	Checksum  uint32 // 4 bytes

	// Data
	Data []byte
}

// Message types
// 0x00 - 0x0F: Connection Management
const (
	MSG_CONN_SYN       MessageType = 0x00 + iota // 0x00
	MGS_CONN_SYN_ACK                             // 0x01
	MSG_CONN_PING                                // 0x02
	MSG_CONN_PONG                                // 0x03
	MSG_CONN_CLOSE                               // 0x04
	MSG_CONN_CLOSE_ACK                           // 0x05
)

// 0x10 - 0x1F: Daemon <-> Server
const (
	MSG_DAEMON_REGISTER          MessageType = 0x10 + iota // 0x10
	MSG_DAEMON_REGISTER_ACK                                // 0x11
	MSG_DAEMON_HEARTBEAT                                   // 0x12
	MSG_DAEMON_HEARTBEAT_ACK                               // 0x13
	MSG_FILE_METADATA_UPLOAD                               // 0x14
	MSG_FILE_METADATA_UPLOAD_ACK                           // 0x15
	MSG_STORAGE_ASSIGN_REQUEST                             // 0x16
	MSG_STORAGE_ASSIGN_RESPONSE                            // 0x17
	MSG_P2P_COORD_REQUEST                                  // 0x18
	MSG_P2P_COORD_RESPONSE                                 // 0x19
)

// 0x20 - 0x2F: Daemon <->  Daemon
const (
	MSG_TRANSFER_INIT         MessageType = 0x20 + iota // 0x20
	MSG_TRANSFER_INIT_ACK                               // 0x21
	MSG_TRANSFER_COMPLETE                               // 0x22
	MSG_TRANSFER_COMPLETE_ACK                           // 0x23
	MSG_CHUNK_DATA                                      // 0x24
	MSG_CHUNK_DATA_ACK                                  // 0x25
	MSG_CHUNK_REQUEST                                   // 0x26
	MSG_CHUNK_REQUEST_NACK                              // 0x27
)

// 0xF0 - 0xFF: Errors and debug
const (
	MSG_DEBUG_MESSAGE          MessageType = 0xF0 + iota // 0xF0
	MSG_ERROR_GENERIC                                    // 0xF1
	MSG_ERROR_AUTH                                       // 0xF2
	MSG_ERROR_NOT_FOUND                                  // 0xF3
	MSG_ERROR_QUOTA_EXCEEDED                             // 0xF4
	MSG_ERROR_RATE_LIMITED                               // 0xF5
	MSG_ERROR_INVALID_SESSION                            // 0xF6
	MSG_ERROR_VERSION_MISMATCH                           // 0xF7
)

// Flags
const (
	FLAG_RESERVED Flags = 1 << iota // 0
	FLAG_PRIO                       // 1
	FLAG_MORE                       // 2
	FLAG_NACK                       // 3
	FLAG_RST                        // 4
	FLAG_FIN                        // 5
	FLAG_ACK                        // 6
	FLAG_SYN                        // 7
)

func Deserialize(data []byte) (error, *Packet) {
	p := &Packet{}
	if len(data) < 20 || len(data) != int(20+p.Length) {
		return fmt.Errorf("packet too short or invalid"), nil
	}
	p.Type = MessageType(data[0])
	p.Flags = Flags(data[1])
	p.Length = uint16(data[2])<<8 | uint16(data[3])
	p.SessionID = uint32(data[4])<<24 | uint32(data[5])<<16 | uint32(data[6])<<8 | uint32(data[7])
	p.SEQNum = uint32(data[8])<<24 | uint32(data[9])<<16 | uint32(data[10])<<8 | uint32(data[11])
	p.ACKNum = uint32(data[12])<<24 | uint32(data[13])<<16 | uint32(data[14])<<8 | uint32(data[15])
	p.Checksum = uint32(data[16])<<24 | uint32(data[17])<<16 | uint32(data[18])<<8 | uint32(data[19])
	p.Data = data[20:]
	return nil, p
}

// Serialize serializes the packet into a byte slice.
func (p *Packet) Serialize() []byte {
	if p.Length == 0 {
		p.Length = uint16(len(p.Data))
	}

	data := make([]byte, 20+len(p.Data))
	data[0] = byte(p.Type)
	data[1] = byte(p.Flags)
	data[2] = byte(p.Length >> 8)
	data[3] = byte(p.Length)
	data[4] = byte(p.SessionID >> 24)
	data[5] = byte(p.SessionID >> 16)
	data[6] = byte(p.SessionID >> 8)
	data[7] = byte(p.SessionID)
	data[8] = byte(p.SEQNum >> 24)
	data[9] = byte(p.SEQNum >> 16)
	data[10] = byte(p.SEQNum >> 8)
	data[11] = byte(p.SEQNum)
	data[12] = byte(p.ACKNum >> 24)
	data[13] = byte(p.ACKNum >> 16)
	data[14] = byte(p.ACKNum >> 8)
	data[15] = byte(p.ACKNum)
	data[16] = byte(p.Checksum >> 24)
	data[17] = byte(p.Checksum >> 16)
	data[18] = byte(p.Checksum >> 8)
	data[19] = byte(p.Checksum)
	copy(data[20:], p.Data)
	return nil
}

// Validate validates the packet.
// TODO: implement MesageType validation
func (p *Packet) Validate() error {
	if p.Length == 0 {
		return fmt.Errorf("packet length is zero")
	}
	if p.Checksum == 0 {
		return fmt.Errorf("packet checksum is zero")
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

func (conn *Connection) NewPacketFromConn(t MessageType, f Flags, data []byte) *Packet {
	p := &Packet{
		Type:      t,
		Flags:     f,
		Length:    uint16(len(data)),
		SessionID: conn.SessionID,
		SEQNum:    conn.LocalSEQNum,
		ACKNum:    conn.RemoteSEQNum + 1,
		Checksum:  0,
		Data:      data,
	}
	p.CalcChecksum()
	return p
}
