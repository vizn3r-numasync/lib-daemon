package dftp

import (
	"fmt"
	"sync"
)

type PacketHander func(*Packet, *Connection) (*Packet, error)

var packetHandlers = map[MessageType]PacketHander{
	MSG_PIGN: HandlePIGN,
	MSG_POGN: HandlePOGN,
}
var m sync.Mutex

var (
	PacketGenericError = fmt.Errorf("an errror has occured")
)

func ErrorPacket(err error) *Packet {
	return &Packet{Type: MSG_ERROR, Data: []byte(err.Error())}
}

func RegisterPacketHandler(t MessageType, h PacketHander) {
	m.Lock()
	packetHandlers[t] = h
	m.Unlock()
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
	c.ConnState = STATE_DATA

	return nil, nil
}

func HandleDATA(p *Packet, c *Connection) (*Packet, error) {
	log.Debug("DATA")
	if c.ConnState != STATE_DATA {
		log.Error("Data packet recieved, but conn is in wrong state")
		return ErrorPacket(PacketGenericError), PacketGenericError
	}

	return nil, nil
}
