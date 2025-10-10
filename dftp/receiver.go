// dftp/manager.go allwos us to have multiple DFTP connections on one UDP socket.

package dftp

import (
	"github.com/vizn3r/go-lib/logger"
	"net"
)

var log *logger.Logger

func init() {
	log = logger.New("DFTP", logger.Cyan)
	log.SetLevel(logger.LevelError)
}

type Receiver struct {
	LocalAddr *net.UDPAddr

	conns map[string]*Connection
	conn  *net.UDPConn
}

func NewReceiver(host string, port int) *Receiver {
	addr := &net.UDPAddr{
		IP:   net.ParseIP(host),
		Port: port,
	}
	return &Receiver{
		LocalAddr: addr,
		conns:     make(map[string]*Connection),
	}
}

func (m *Receiver) NewConnection(addr *net.UDPAddr) *Connection {
	conn := NewEmptyConnection()
	conn.RemoteAddr = addr
	conn.LocalAddr = m.LocalAddr
	conn.conn = m.conn
	conn.ConnState = STATE_IDLE
	return conn

}

func (m *Receiver) GetConnection(addr *net.UDPAddr) *Connection {
	return m.conns[addr.String()]
}

func (m *Receiver) CloseConnection(addr *net.UDPAddr) {
	m.conns[addr.String()].Close()
	delete(m.conns, addr.String())
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
		buf := make([]byte, PACKET_SIZE)
		n, addr, err := m.conn.ReadFromUDP(buf)
		if err != nil {
			log.Error("Error reading from UDP socket: ", err)
			continue
		}
		if n == 0 {
			continue
		}
		conn, ok := m.conns[addr.String()]
		if !ok {
			conn = m.NewConnection(addr)
			conn.conn = m.conn
			conn.RemoteAddr = addr
			m.conns[addr.String()] = conn
		}
		go conn.handleReq(buf[:n])
	}

}

func (m *Receiver) Close() {
	for _, conn := range m.conns {
		go conn.Close()
	}
	delete(m.conns, "")
}
