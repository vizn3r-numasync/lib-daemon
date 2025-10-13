package dftp

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/vizn3r/go-lib/logger"
)

type PacketHander func(*Packet, *Connection) (*Packet, error)

type DataHandler struct {
	SessionID uint32

	expectedInitPackets uint32
	receivedInitPackets uint32
	initBuf             []byte

	ChunkMap map[uint32]uint32
}

var packetHandlers = map[MessageType]PacketHander{
	MSG_PIGN: HandlePIGN,
	MSG_POGN: HandlePOGN,

	MSG_TRANSFER_INIT: HandleTRANSFER_INIT,

	MSG_ERROR: HandleERROR,
}

var dataHandlers = map[uint32]*DataHandler{}

var m sync.Mutex

var handl = logger.New("HNDL", logger.Red)

var (
	PacketGenericError        = fmt.Errorf("an unexpected errror has occured")
	PacketWrongStateError     = fmt.Errorf("conn is in wrong state")
	PacketInvalidAddressError = fmt.Errorf("invalid address")
)

func NewDataHandler(sessionID uint32) *DataHandler {
	dh := &DataHandler{
		SessionID:           sessionID,
		initBuf:             make([]byte, DATA_SIZE+4),
		expectedInitPackets: 1,
		receivedInitPackets: 0,
		ChunkMap:            nil, // dont need to allocate, will be filled in after TRANSFER_INIT
	}
	dataHandlers[sessionID] = dh
	return dh
}

// handlePacket handles a packet received from the Connection.RemoteAddr.
func (conn *Connection) handlePacket(packet *Packet) (*Packet, error) {
	handl.Debug("Received type: ", packet.Type)
	handler, ok := packetHandlers[packet.Type]
	if !ok {
		err := fmt.Errorf("No handler for packet type %d", packet.Type)
		handl.Error("Error handling packet: ", err)
		return &Packet{
			Type: MSG_ERROR,
			Data: []byte(err.Error()),
		}, err
	}
	packet, err := handler(packet, conn)
	if err != nil {
		handl.Error("Error handling packet: ", err)
		return &Packet{
			Type: MSG_ERROR,
			Data: []byte(err.Error()),
		}, err
	}
	return packet, nil
}

func ErrorPacket(err error) *Packet {
	return &Packet{Type: MSG_ERROR, Data: []byte(err.Error())}
}

func RegisterPacketHandler(t MessageType, h PacketHander) {
	m.Lock()
	packetHandlers[t] = h
	m.Unlock()
}

func HandleERROR(p *Packet, c *Connection) (*Packet, error) {
	handl.Debug("ERROR")
	handl.Error("Error recieved: ", string(p.Data))
	return nil, nil
}

func HandlePIGN(p *Packet, c *Connection) (*Packet, error) {
	handl.Debug("PIGN!")
	return &Packet{Type: MSG_POGN, Data: []byte("POGN!")}, nil
}

func HandlePOGN(p *Packet, c *Connection) (*Packet, error) {
	handl.Debug("POGN!")
	return nil, nil
}

// Data stuff

func HandleTRANSFER_INIT(p *Packet, c *Connection) (*Packet, error) {
	handl.Debug("TRANSFER_INIT")

	dh, ok := dataHandlers[c.SessionID]
	if !ok {
		dh = NewDataHandler(c.SessionID)
	}

	if p.ChunkNum == 0 {
		numPackets := binary.LittleEndian.Uint32(p.Data[0:4])
		dh.expectedInitPackets = numPackets
		dh.initBuf = append(dh.initBuf, p.Data[4:]...)
	} else {
		dh.initBuf = append(dh.initBuf, p.Data...)
	}

	if dh.expectedInitPackets >= dh.receivedInitPackets {
		chunks := NewChunks()
		if err := chunks.DeserializeMap(dh.initBuf); err != nil {
			handl.Error("Error deserializing chunks: ", err)
			return ErrorPacket(err), err
		}
		dh.ChunkMap = chunks.ChunkMap()
		c.State = STATE_DATA
	}

	return nil, nil
}

func HandleDATA(p *Packet, c *Connection) (*Packet, error) {
	handl.Debug("DATA")
	if c.State != STATE_DATA {
		handl.Error("Data packet recieved, but conn is in wrong state")
		return ErrorPacket(PacketWrongStateError), PacketWrongStateError
	}

	// TODO: IMPLEMENT

	return nil, nil
}
