// dftp/sender.go manages the sending of data and multiplexing of connections
package dftp

import (
	"encoding/binary"
	"hash/fnv"
	"net"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/vizn3r/go-lib/logger"
)

type Sender struct {
	State      ConnState
	RemoteAddr *net.UDPAddr
	SessionID  uint32

	Connections map[uint8]*Connection

	maxConcurrent uint8

	wg sync.WaitGroup
	mu sync.RWMutex
}

func NewEmptySender() *Sender {
	return &Sender{
		RemoteAddr:    nil,
		SessionID:     0,
		Connections:   make(map[uint8]*Connection),
		maxConcurrent: 0,
	}
}

var sendl = logger.New("SNDR", logger.Blue)

func NewSession(ip string, port int, sessionID uint32, maxConcurrent uint8) *Sender {
	if maxConcurrent < 1 {
		sendl.Warn("maxConcurrent must be at least 1, setting to 1")
		maxConcurrent = 1
	}
	s := NewEmptySender()
	s.RemoteAddr = &net.UDPAddr{IP: net.ParseIP(ip), Port: port}
	s.maxConcurrent = maxConcurrent

	if sessionID == 0 {
		for {
			uuid := uuid.Must(uuid.NewV4())
			h := fnv.New32a()
			h.Write(uuid[:])
			if _, ok := usedSessionIDs.Load(h.Sum32()); ok {
				continue
			}
			s.SessionID = h.Sum32()
			usedSessionIDs.Store(s.SessionID, true)
			break
		}
	} else {
		s.SessionID = sessionID
	}

	sendl.Debug("Creating ", maxConcurrent, " connections for ", ip, ":", port)
	for i := range s.maxConcurrent {
		conn, err := NewConn(ip, 0, s.SessionID)
		if err != nil {
			sendl.Error("Error creating connection: ", err)
			return nil
		}

		conn.SessionID = s.SessionID
		conn.ConnID = uint8(i)
		conn.RemoteAddr = s.RemoteAddr

		sendl.Debug("Created connection connID: ", conn.ConnID, " for ", ip, ":", port, " sessionID: ", s.SessionID)
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
	chunkNum := uint32((len(data) + DATA_SIZE - 1) / DATA_SIZE)
	chunks := make([]*Chunk, chunkNum)
	for i := range chunkNum {
		chunks[i] = &Chunk{
			ID:       i,
			Checksum: 0,
			Data:     data[i*DATA_SIZE : min((i+1)*DATA_SIZE, uint32(len(data)))],
			Received: false,
		}
	}

	sendl.Debug("Preparing ", len(data), " bytes for ", s.RemoteAddr, " seessionID: ", s.SessionID)

	if s.State != STATE_DATA {
		sendl.Debug("Preparing chunks")

		s.forEachConn(func(i uint8, conn *Connection) error {
			chunkMap := []byte{} // uint32 chunkID + uint32 checksum

			// Populate chunkMap based on number of streams
			offset := 0
			for chunkID := uint32(i); chunkID < chunkNum; chunkID += uint32(s.maxConcurrent) {
				chunkMap = append(chunkMap, make([]byte, 8)...)
				binary.LittleEndian.PutUint32(chunkMap[offset:offset+4], chunkID)
				binary.LittleEndian.PutUint32(chunkMap[offset+4:offset+8], chunks[chunkID].Checksum)
				offset += 8
			}

			numPackets := (len(chunkMap) + PACKET_SIZE - 1) / PACKET_SIZE
			sendl.Debug("Number of packets: ", numPackets)
			for i := range numPackets {
				start := i * PACKET_SIZE
				end := min(start+PACKET_SIZE, len(chunkMap))
				packet := &Packet{
					Type:     MSG_TRANSFER_INIT,
					ChunkNum: uint32(i),
					Data:     chunkMap[start:end],
				}

				if i == 0 {
					buf := make([]byte, 4+len(packet.Data))
					binary.LittleEndian.PutUint32(buf, packet.ChunkNum)
					copy(buf[4:], packet.Data)
					packet.Data = buf
				}

				sendl.Debug("Sending chunkMap packetID: ", i, " to ", conn.RemoteAddr, " sessionID: ", conn.SessionID, " connID: ", conn.ConnID)
				if err := conn.Send(packet); err != nil {
					sendl.Error("Error sending transfer init: ", err)
					return err
				}
			}
			return nil
		})

		s.State = STATE_DATA
	}

	// Send data
	s.forEachConn(func(i uint8, conn *Connection) error {
		go func(i uint8, conn *Connection) {
			for {
				var chunk *Chunk = nil
				s.mu.Lock()
				for chunkID := uint32(i); chunkID < chunkNum; chunkID += uint32(s.maxConcurrent) {
					if !chunks[chunkID].Received {
						chunk = chunks[chunkID]
						chunks[chunkID].Received = true
						break
					}
				}
				s.mu.Unlock()

				if chunk == nil {
					data := &Packet{
						Type: MSG_CHECK,
						Data: nil,
					}
					if err := conn.Send(data); err != nil {
						sendl.Error("Error sending check: ", err)
						return
					}
					break
				}
				data := &Packet{
					Type: MSG_DATA,
					Data: chunk.Data,
				}

				sendl.Debug("Sending data packetID: ", i, " to ", conn.RemoteAddr, " sessionID: ", conn.SessionID, " connID: ", conn.ConnID)
				if err := conn.Send(data); err != nil {
					sendl.Error("Error sending data: ", err)
					return
				}
			}
		}(i, conn)
		return nil
	})
	s.State = STATE_IDLE
}

func (s *Sender) Recv() {
	s.forEachConn(func(i uint8, conn *Connection) error {
		s.wg.Add(1)
		go func(i uint8, conn *Connection) {
			conn.Recv()
			s.wg.Done()
		}(i, conn)
		return nil
	})
	s.wg.Wait()
}
