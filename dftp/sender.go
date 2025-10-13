// dftp/sender.go manages the sending of data and multiplexing of connections
package dftp

import (
	"encoding/binary"
	"hash/fnv"
	"net"

	"github.com/gofrs/uuid"
	"github.com/vizn3r/go-lib/logger"
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
	chunks := NewChunkMap()
	chunkMaps := NewChunksMap()

	sendl.Debug("Preparing ", len(data), " bytes for ", s.RemoteAddr, " seessionID: ", s.SessionID)

	if s.State != STATE_DATA {
		// Prepare chunks
		sendl.Debug("Preparing chunks")
		for chunkID := range uint32(len(data) / DATA_SIZE) {
			chunks[chunkID] = &Chunk{
				ID:       chunkID,
				Checksum: 0,
				Data:     data[chunkID*DATA_SIZE : (chunkID+1)*DATA_SIZE],
				received: false,
			}
		}

		sendl.Debug("Preparing chunk maps")
		chunkMaps = ChunkIDMaptoStreamMap(chunks, s.maxConcurrent)

		s.forEachConn(func(i uint8, conn *Connection) error {
			chunkMapData := chunkMaps[i].SerializeMap()
			numPackets := (len(chunkMapData) + PACKET_SIZE - 1) / PACKET_SIZE
			sendl.Debug("Number of packets: ", numPackets)
			for i := range numPackets {
				start := i * PACKET_SIZE
				end := min(start+PACKET_SIZE, len(chunkMapData))
				packet := &Packet{
					Type:     MSG_TRANSFER_INIT,
					ChunkNum: uint32(i),
					Data:     chunkMapData[start:end],
				}

				if i == 0 {
					buf := make([]byte, 4+len(packet.Data))
					binary.LittleEndian.PutUint32(buf, packet.ChunkNum)
					copy(buf[4:], packet.Data)
					packet.Data = buf
				}

				sendl.Debug("Sending packetID: ", i, " to ", conn.RemoteAddr, " sessionID: ", conn.SessionID, " connID: ", conn.ConnID)
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
				chunk := chunkMaps[i].Next()
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

				sendl.Debug("Sending packetID: ", i, " to ", conn.RemoteAddr, " sessionID: ", conn.SessionID, " connID: ", conn.ConnID)
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
