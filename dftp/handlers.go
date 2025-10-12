package dftp

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type PacketHander func(*Packet, *Connection) (*Packet, error)

var packetHandlers = map[MessageType]PacketHander{
	MSG_PIGN: HandlePIGN,
	MSG_POGN: HandlePOGN,

	MSG_ERROR: HandleERROR,
}
var m sync.Mutex

var (
	PacketGenericError = fmt.Errorf("an errror has occured")
)

// handlePacket handles a packet received from the Connection.RemoteAddr.
func (conn *Connection) handlePacket(packet *Packet) (*Packet, error) {
	log.Debug("Received type: ", packet.Type)
	handler, ok := packetHandlers[packet.Type]
	if !ok {
		err := fmt.Errorf("No handler for packet type %d", packet.Type)
		log.Error("Error handling packet: ", err)
		return &Packet{
			Type: MSG_ERROR,
			Data: []byte(err.Error()),
		}, err
	}
	packet, err := handler(packet, conn)
	if err != nil {
		log.Error("Error handling packet: ", err)
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
	log.Debug("ERROR")
	log.Error("Error recieved: ", string(p.Data))
	return nil, nil
}

func HandlePIGN(p *Packet, c *Connection) (*Packet, error) {
	log.Debug("PIGN!")
	return &Packet{Type: MSG_POGN, Data: []byte("POGN!")}, nil
}

func HandlePOGN(p *Packet, c *Connection) (*Packet, error) {
	log.Debug("POGN!")
	return nil, nil
}

func HandleTRANSFER_INIT(p *Packet, c *Connection) (*Packet, error) {
	log.Debug("TRANSFER_INIT")

	rawList := string(p.Data)
	chunkData := strings.SplitSeq(rawList, ";")
	for chunk := range chunkData {
		ch := strings.Split(chunk, ":")
		id, err := strconv.ParseUint(ch[0], 10, 32)
		chunkID := uint32(id)
		if err != nil {
			log.Error("Error parsing chunk id: ", err)
			return ErrorPacket(err), err
		}
		checksum, err := strconv.ParseUint(ch[1], 10, 32)
		chunkChecksum := uint32(checksum)
		if err != nil {
			log.Error("Error parsing chunk checksum: ", err)
			return ErrorPacket(err), err
		}

		c.chunkMap[chunkID] = &Chunk{
			ID:       chunkID,
			Checksum: chunkChecksum,
			Data:     []byte{},
			received: false,
		}
	}

	c.State = STATE_DATA

	return nil, nil
}

func HandleDATA(p *Packet, c *Connection) (*Packet, error) {
	log.Debug("DATA")
	if c.State != STATE_DATA {
		log.Error("Data packet recieved, but conn is in wrong state")
		return ErrorPacket(PacketGenericError), PacketGenericError
	}

	return nil, nil
}
