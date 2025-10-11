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
		conn, err := NewConn(ip, 0) // O so the port is random
		if err != nil {
			log.Error("Error creating connection: ", err)
			return nil
		}
		s.Connections[i] = conn
	}

	s.State = STATE_IDLE
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
				chunkMapData.WriteString(";")
			}
			init := &Packet{
				Type: MSG_TRANSFER_INIT,
				Data: chunkMapData.Bytes(),
			}
			conn.Send(init)
		})
		s.State = STATE_DATA
	}

	// Send data
	s.forEachConn(func(i uint8, conn *Connection) {
		connChunks, ok := chunkMaps[i]
		if !ok || len(connChunks) == 0 {
			conn.Send(&Packet{
				Type:      MSG_COMPLETE,
				SessionID: s.SessionID,
				StreamID:  i,
			})
			return
		}

		chunk := connChunks[0]
		chunkMaps[i] = connChunks[1:]

		conn.Send(&Packet{
			Type:      MSG_DATA,
			Data:      chunk.Data,
			Checksum:  chunk.Checksum,
			SessionID: s.SessionID,
			StreamID:  i,
		})
	})
	s.State = STATE_IDLE
}
