package dftp

import (
	"sync"
)

var packetHandlers = map[MessageType]PacketHander{}
var m sync.RWMutex

func HandlersInit() {
	RegisterPacketHandler(MSG_PIGN, HandlePIGN)
	RegisterPacketHandler(MSG_POGN, HandlePOGN)
}

func RegisterPacketHandler(t MessageType, h PacketHander) {
	m.Lock()
	packetHandlers[t] = h
	m.Unlock()
}

func HandlePIGN(p *Packet) (*Packet, error) {
	log.Debug("PIGN!")
	return &Packet{Type: MSG_POGN, Data: []byte("POGN!")}, nil
}

func HandlePOGN(p *Packet) (*Packet, error) {
	log.Debug("POGN!")
	return nil, nil
}
