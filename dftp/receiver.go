// dftp/receiver.go allwos us to have multiple DFTP connections on one UDP socket.

package dftp

import (
	"net"
	"sync"

	"github.com/vizn3r/go-lib/logger"
)

var log *logger.Logger

func init() {
	log = logger.New("DFTP", logger.Cyan)
	//log.SetLevel(logger.LevelError)
}

type Receiver struct {
	LocalAddr *net.UDPAddr

	conns map[uint32]*RecvConn
	conn  *net.UDPConn

	mu sync.RWMutex
}

type RecvConn struct {
	*Connection
	streams map[uint8]*Connection
}

func NewReceiver(host string, port int) *Receiver {
	addr := &net.UDPAddr{
		IP:   net.ParseIP(host),
		Port: port,
	}
	return &Receiver{
		LocalAddr: addr,
		conns:     make(map[uint32]*RecvConn),
	}
}

func (m *Receiver) NewRecvStream(addr *net.UDPAddr, sessionID uint32, streamID uint8) *Connection {
	conn := NewEmptyConnection()

	log.Debug("Creating new stream - addr: ", addr, " sessionID: ", sessionID, " streamID: ", streamID)

	conn.RemoteAddr = addr
	conn.LocalAddr = m.LocalAddr
	conn.SessionID = sessionID
	conn.ConnID = streamID
	conn.State = STATE_IDLE
	conn.conn = m.conn

	return conn

}

func (m *Receiver) NewRecvConn() *RecvConn {
	return &RecvConn{
		Connection: NewEmptyConnection(),
		streams:    make(map[uint8]*Connection),
	}
}

func (m *Receiver) GetConnections(sessionID uint32) *RecvConn {
	return m.conns[sessionID]
}

func (m *Receiver) CloseConnection(sessionID uint32) {
	m.forEachConn(sessionID, func(conn *Connection) error {
		conn.Close()
		return nil
	})
	delete(m.conns, sessionID)
}

func (m *Receiver) CloseAllConnections() {
	for sessionID, conn := range m.conns {
		for streamID, stream := range conn.streams {
			delete(conn.streams, streamID)
			stream.CloseWithoutConn()
		}
		delete(m.conns, sessionID)
	}
}

var recvBufferPool = sync.Pool{
	New: func() any {
		return make([]byte, PACKET_SIZE)
	},
}

func (m *Receiver) Listen(ready chan<- struct{}) (err error) {
	m.conn, err = net.ListenUDP("udp", m.LocalAddr)
	if err != nil {
		return err
	}

	log.Info("Listening on ", m.LocalAddr)

	if ready != nil {
		ready <- struct{}{}
	}

	for {
		bufPtr := recvBufferPool.Get().([]byte)
		n, addr, err := m.conn.ReadFromUDP(bufPtr)
		if err != nil {
			log.Error("Error reading from UDP socket: ", err)
			continue
		}

		if n == 0 {
			log.Error("Received empty packet")
			continue
		}

		packet, err := Deserialize(bufPtr[:n])
		recvBufferPool.Put(bufPtr)
		if err != nil {
			log.Error("Error deserializing packet: ", err)
			continue
		}

		sessionID := packet.SessionID
		streamID := packet.StreamID

		log.Debug("Received packet - SessionID: ", sessionID, " StreamID: ", streamID, " Type: ", packet.Type)

		if sessionID == 0 {
			log.Error("Invalid session ID: ", sessionID)
			continue
		}

		m.mu.RLock()
		recvConn, ok := m.conns[sessionID]
		m.mu.RUnlock()
		if !ok {
			m.mu.Lock()
			recvConn = m.NewRecvConn()
			m.conns[sessionID] = recvConn
			m.mu.Unlock()
		}

		m.mu.RLock()
		stream, ok := recvConn.streams[streamID]
		m.mu.RUnlock()
		if !ok {
			m.mu.Lock()
			stream = m.NewRecvStream(addr, sessionID, streamID)
			stream.SessionID = sessionID
			m.conns[sessionID].streams[streamID] = stream
			m.mu.Unlock()
		}

		// Check remoteAddr
		if stream.RemoteAddr.String() != addr.String() {
			log.Error("Remote address mismatch")
			stream.Send(ErrorPacket(PacketGenericError))
			continue
		}

		go stream.handleReq(packet)
	}
}

func (m *Receiver) forEachConn(sessionID uint32, f func(*Connection) error) error {
	// TODO: implement
	return nil
}

func (conn *Connection) handleReq(packet *Packet) {
	log.Debug("Received packet from ", conn.RemoteAddr, " type: ", packet.Type, " data: ", string(packet.Data))

	resp, err := conn.handlePacket(packet)
	if err != nil {
		log.Error("Error handling packet: ", err)
		return
	}
	conn.Send(resp)
}

func (m *Receiver) Close() {
	for _, conn := range m.conns {
		for _, stream := range conn.streams {
			stream.Close()
		}
	}
	// TODO: close conn
}
