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

type ConnManager struct {
	LocalAddr *net.UDPAddr

	conns map[string]*Connection
	conn  *net.UDPConn
}

func NewConnManager(host string, port int) *ConnManager {
	HandlersInit()
	addr := &net.UDPAddr{
		IP:   net.ParseIP(host),
		Port: port,
	}
	return &ConnManager{
		LocalAddr: addr,
		conns:     make(map[string]*Connection),
	}
}

func (m *ConnManager) NewConnection(addr *net.UDPAddr) *Connection {
	conn := NewEmptyConnection()
	conn.RemoteAddr = addr
	conn.LocalAddr = m.LocalAddr
	conn.conn = m.conn
	conn.ConnState = STATE_IDLE
	return conn

}

func (m *ConnManager) GetConnection(addr *net.UDPAddr) *Connection {
	return m.conns[addr.String()]
}

func (m *ConnManager) CloseConnection(addr *net.UDPAddr) {
	m.conns[addr.String()].Close()
	delete(m.conns, addr.String())
}

func (m *ConnManager) Listen() (err error) {
	m.conn, err = net.ListenUDP("udp", m.LocalAddr)
	if err != nil {
		return err
	}

	log.Info("Listening on ", m.LocalAddr)

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

func (m *ConnManager) Close() {
	for _, conn := range m.conns {
		go conn.Close()
	}
}
