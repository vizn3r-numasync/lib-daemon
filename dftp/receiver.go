// dftp/receiver.go allwos us to have multiple DFTP connections on one UDP socket.

package dftp

import (
	"net"
	"strconv"
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

	conns sync.Map
	conn  *net.UDPConn

	mu sync.RWMutex
}

type RecvConn struct {
	*Connection
	streams sync.Map
}

func NewReceiver(host string, port int) *Receiver {
	addr := &net.UDPAddr{
		IP:   net.ParseIP(host),
		Port: port,
	}
	return &Receiver{
		LocalAddr: addr,
		conns:     sync.Map{},
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
		streams:    sync.Map{},
	}
}

func (m *Receiver) GetConnections(sessionID uint32) *RecvConn {
	conn, ok := m.conns.Load(sessionID)
	if !ok {
		return nil
	}
	return conn.(*RecvConn)
}

func (m *Receiver) CloseConnection(sessionID uint32) {
	// TODO: implement
}

func (m *Receiver) CloseAllConnections() {
	// Turns out I don't need to do it C style :P
	m.conns.Clear()
}

var recvBufferPool = sync.Pool{
	New: func() any {
		b := make([]byte, PACKET_SIZE)
		return &b
	},
}

var workerPool = make(chan struct{}, 1024)

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
		bufPtr := recvBufferPool.Get().(*[]byte)
		buf := *bufPtr
		n, addr, err := m.conn.ReadFromUDP(buf)
		if err != nil {
			log.Error("Error reading from UDP socket: ", err)
			continue
		}

		if n == 0 {
			log.Error("Received empty packet")
			continue
		}

		packet, err := Deserialize(buf[:n])
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

		v, ok := m.conns.Load(sessionID)
		if !ok {
			v = m.NewRecvConn()
			m.conns.Store(sessionID, v)
		}
		recvConn := v.(*RecvConn)

		var stream *Connection
		v, ok = recvConn.streams.Load(streamID)
		if !ok {
			stream = m.NewRecvStream(addr, sessionID, streamID)
			stream.SessionID = sessionID
			recvConn.streams.Store(streamID, stream)
		} else {
			stream = v.(*Connection)
		}

		// Check remoteAddr
		if !addr.IP.Equal(stream.RemoteAddr.IP) || addr.Port != stream.RemoteAddr.Port {
			log.Error("Remote address mismatch for "+strconv.Itoa(int(stream.SessionID))+" expected ", stream.RemoteAddr, " got ", addr)
			stream.Send(ErrorPacket(PacketGenericError))
			continue
		}

		go stream.handleReq(packet)
	}
}

func (m *Receiver) forEachConn(sessionID uint32, f func(*Connection) error) error {
	v, ok := m.conns.Load(sessionID)
	if !ok {
		return nil
	}
	conn := v.(*RecvConn)
	conn.streams.Range(func(key, value any) bool {
		if err := f(value.(*Connection)); err != nil {
			return false
		}
		return true
	})
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
	m.conns.Range(func(key, value any) bool {
		value.(*RecvConn).streams.Range(func(key, value any) bool {
			value.(*Connection).Close()
			return true
		})
		value.(*RecvConn).streams.Clear()
		usedSessionIDs.Delete(key)
		return true
	})
	m.conns.Clear()
}
