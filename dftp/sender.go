// dftp/sender.go manages the sending of data and multiplexing of connections
package dftp

import (
	"bytes"
	"encoding/binary"
	"net"
)

type Sender struct {
	State      ConnState
	RemoteAddr *net.UDPAddr
	SessionID  uint32

	Connections map[uint8]*Connection

	maxConcurrent uint8
}

type Chunk struct {
	ID       uint32
	Checksum uint32
	Data     []byte
}

func NewEmptySender() *Sender {
	return &Sender{
		RemoteAddr:    nil,
		SessionID:     0,
		Connections:   make(map[uint8]*Connection),
		maxConcurrent: 0,
	}
}

func NewSession(ip string, port int, maxConcurrent uint8) *Sender {
	if maxConcurrent < 1 {
		log.Warn("maxConcurrent must be at least 1, setting to 1")
		maxConcurrent = 1
	}
	s := NewEmptySender()
	s.RemoteAddr = &net.UDPAddr{IP: net.ParseIP(ip), Port: port}
	s.maxConcurrent = maxConcurrent

	for i := range s.maxConcurrent {
		conn, err := NewConn(ip, port)
		if err != nil {
			log.Error("Error creating connection: ", err)
			return nil
		}
		s.Connections[i] = conn
	}
	return s
}

func (s *Sender) forEachConn(f func(i uint8, conn *Connection)) {
	for j, c := range s.Connections {
		f(j, c)
	}
}

func (s *Sender) Send(data []byte) {
	chunks := map[uint32]Chunk{}
	chunkMaps := map[uint8][]Chunk{}
	if s.State != STATE_DATA {
		// Prepare chunks
		for chunkID := range uint32(len(data) / DATA_SIZE) {
			chunks[chunkID] = Chunk{
				ID:       chunkID,
				Checksum: 0,
				Data:     data[chunkID*DATA_SIZE : (chunkID+1)*DATA_SIZE],
			}
		}

		// Make chunkmaps
		for i := range chunks {
			connID := uint8(i) % s.maxConcurrent
			chunkMaps[connID] = append(chunkMaps[connID], chunks[i])
		}

		s.forEachConn(func(i uint8, conn *Connection) {
			chunkMapData := bytes.NewBuffer([]byte{})
			for _, chunk := range chunkMaps[i] {
				binary.Write(chunkMapData, binary.LittleEndian, chunk.ID)
				chunkMapData.WriteString(":")
				binary.Write(chunkMapData, binary.LittleEndian, chunk.Checksum)
				chunkMapData.WriteString(":")
			}
			init := &Packet{
				Type: MSG_TRANSFER_INIT,
				Data: chunkMapData.Bytes(),
			}
			conn.Send(init)
		})
	}
}
