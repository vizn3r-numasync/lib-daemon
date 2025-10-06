package protocol

import (
	"net"
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
}

const (
	CONN_CLOSED ConnectionState = iota
	CONN_SYN_SENT
	CONN_ESTABLISHED
	CONN_FIN_WAIT
)

func (conn *Connection) Send(packet *Packet) (err error) {
	buf := packet.Serialize()
	_, err = conn.conn.WriteToUDP(buf, conn.RemoteAddr)
	if err != nil {
		return err
	}
	return nil
}

// UDP Listener
func UDPListen(ip string, port int) (err error) {
	conn := Connection{
		State:        CONN_CLOSED,
		LocalSEQNum:  0,
		RemoteSEQNum: 0,
		SessionID:    0,
		RemoteAddr:   nil,
		LocalAddr:    nil,
	}

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
	for {
		_, conn.RemoteAddr, err = conn.conn.ReadFromUDP(buf)
		if err != nil {
			return err
		}

		err, packet := Deserialize(buf)
		if err != nil {
			return err
		}

		switch conn.State {
		case CONN_CLOSED:
			err = conn.OpenConnection(packet)
			if err != nil {
				return err
			}
		}
	}
}

func (conn *Connection) OpenConnection(packet *Packet) (err error) {
	if packet.Type == MSG_CONN_SYN && packet.Flags == FLAG_SYN {
		// later, payload will be version + status
		data := conn.NewPacketFromConn(MSG_CONN_SYN, FLAG_ACK, nil)
		data.SEQNum = conn.SessionID
		data.ACKNum = conn.LocalSEQNum
		conn.Send(data)
	}

	return nil
}
