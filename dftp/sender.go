// dftp/sender.go manages the sending of data and multiplexing of connections
package dftp

import (
	"net"
)

type Sender struct {
	State      ConnState
	RemoteAddr *net.UDPAddr
	SessionID  uint32

	Connections map[uint8]*Connection

	maxConcurrent uint8
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
		conn, err := NewConn(ip, 0, 0) // O so the port is random
		conn.SessionID = s.SessionID
		conn.ConnID = uint8(i)
		if err != nil {
			log.Error("Error creating connection: ", err)
			return nil
		}
		s.Connections[i] = conn
	}

	s.State = STATE_IDLE
	return s
}

func (s *Sender) forEachConn(f func(i uint8, conn *Connection) error) error {
	for j, c := range s.Connections {
		if err := f(j, c); err != nil {
			return err
		}
	}
	return nil
}

func (s *Sender) Send(data []byte) {
	chunks := NewChunkIDMap()
	chunkMaps := NewChunkStreamMap()

	if s.State != STATE_DATA {
		// Prepare chunks
		for chunkID := range uint32(len(data) / DATA_SIZE) {
			chunks[chunkID] = &Chunk{
				ID:       chunkID,
				Checksum: 0,
				Data:     data[chunkID*DATA_SIZE : (chunkID+1)*DATA_SIZE],
				received: false,
			}
		}

		chunkMaps = ChunkIDMaptoStreamMap(chunks, s.maxConcurrent)

		s.forEachConn(func(i uint8, conn *Connection) error {
			chunkMapData := chunkMaps[i].Serialize()
			init := &Packet{
				Type: MSG_TRANSFER_INIT,
				Data: chunkMapData,
			}
			return conn.Send(init)
		})

		s.State = STATE_DATA
	}

	// Send data
	s.forEachConn(func(i uint8, conn *Connection) error {
		go func(i uint8, conn *Connection) {
			for {
				chunk := chunkMaps[i].Next()
				if chunk == nil {
					data := &Packet{
						Type: MSG_CHECK,
						Data: nil,
					}
					if err := conn.Send(data); err != nil {
						log.Error("Error sending check: ", err)
						return
					}
					break
				}
				data := &Packet{
					Type: MSG_DATA,
					Data: chunk.Data,
				}
				if err := conn.Send(data); err != nil {
					log.Error("Error sending data: ", err)
					return
				}
			}
		}(i, conn)
		return nil
	})
	s.State = STATE_IDLE
}
