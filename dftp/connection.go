package dftp

import (
	"fmt"
	"log"
	"net"
	"time"
)

type ConnectionState int

type Connection struct {
	State        ConnectionState
	LocalSEQNum  uint32
	RemoteSEQNum uint32
	SessionID    uint32
	LocalAddr    *net.UDPAddr
	RemoteAddr   *net.UDPAddr
	conn         *net.UDPConn
	packet       chan *Packet
}

const (
	CONN_CLOSED ConnectionState = iota
	CONN_SYN_SENT
	CONN_ESTABLISHED
	CONN_FIN_WAIT
)

func NewEmptyConnection() *Connection {
	return &Connection{
		State:        CONN_CLOSED,
		LocalSEQNum:  0,
		RemoteSEQNum: 0,
		SessionID:    0,
		LocalAddr:    nil,
		RemoteAddr:   nil,
		conn:         nil,
		packet:       make(chan *Packet, 100),
	}
}

func Dial(ip string, port int) (conn *Connection, err error) {
	conn = NewEmptyConnection()

	log.Println("Dialing", ip, port)

	conn.RemoteAddr = &net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	}

	conn.conn, err = net.ListenUDP("udp", conn.LocalAddr)
	if err != nil {
		return nil, err
	}

	conn.LocalAddr = conn.conn.LocalAddr().(*net.UDPAddr)

	go conn.receiver()

	return conn, nil
}

func (conn *Connection) Close() error {
	if conn.conn != nil {
		close(conn.packet)
		return conn.conn.Close()
	}
	return nil
}

func (conn *Connection) Send(packet *Packet) error {
	buf := packet.Serialize()

	log.Println("Sending from", conn.LocalAddr, "to", conn.RemoteAddr, "bytes:", len(buf))

	_, err := conn.conn.WriteToUDP(buf, conn.RemoteAddr)
	if err != nil {
		return err
	}

	return nil
}

func (conn *Connection) receiver() {
	buf := make([]byte, PACKET_SIZE)
	for {
		n, remoteAddr, err := conn.conn.ReadFromUDP(buf)
		if err != nil {
			log.Println("Error receiving from", conn.RemoteAddr, "bytes:", n)
			return
		}
		conn.RemoteAddr = remoteAddr
		log.Println("Received", conn.RemoteAddr, "bytes:", n)

		err, packet := Deserialize(buf[:n])
		if err != nil {
			log.Println("Error deserializing packet", err)
			return
		}

		log.Println("Received", string(packet.Data))

		select {
		case conn.packet <- packet:
		default:
			log.Println("Packet queue full")
		}
	}
}

func (conn *Connection) Receive(expType MessageType, timeout time.Duration) (*Packet, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case packet := <-conn.packet:
			if packet.Type == expType {
				return packet, nil
			}
			log.Println("Received unexpected packet", packet.Type)
		case <-timer.C:
			return nil, fmt.Errorf("Timeout")
		}
	}
}

// UDP Listener
func UDPListen(ip string, port int) (err error) {
	conn := NewEmptyConnection()

	conn.LocalAddr = &net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	}

	conn.conn, err = net.ListenUDP("udp", conn.LocalAddr)
	if err != nil {
		return err
	}
	defer conn.conn.Close()

	buf := make([]byte, PACKET_SIZE)
	var n int
	for {
		n, conn.RemoteAddr, err = conn.conn.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		log.Println("Received", conn.RemoteAddr, "bytes:", n)

		err, packet := Deserialize(buf[:n])
		if err != nil {
			return err
		}

		log.Println("Received", string(packet.Data))

		switch packet.Type {
		case MSG_CONN_PING:
			log.Println("Received PIGN!, responding with POGN!")
			conn.Send(&Packet{
				Type: MSG_CONN_PONG,
				Data: []byte("POGN!"),
			})
		}
	}
}
