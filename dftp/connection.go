// dftp/connection.go handles the connection between two DFTP nodes.
// It handles packet sending and receiving, and the packet channel.

package dftp

import (
	"fmt"
	"net"
	"time"
)

type ConnectionState int

type Connection struct {
	Handler      *ConnHandler
	LocalSEQNum  uint32
	RemoteSEQNum uint32
	SessionID    uint32
	LocalAddr    *net.UDPAddr
	RemoteAddr   *net.UDPAddr

	conn   *net.UDPConn
	packet chan *Packet
}

//------------------------------------------------------------------------------
//
// Connection creation / destruction
//
//------------------------------------------------------------------------------

// NewEmptyConnection creates a new empty Connection struct.
// Use NewConn() to create a connection.
func NewEmptyConnection() *Connection {
	return &Connection{
		Handler:      nil,
		LocalSEQNum:  0,
		RemoteSEQNum: 0,
		SessionID:    0,
		LocalAddr:    nil,
		RemoteAddr:   nil,
		conn:         nil,
		packet:       make(chan *Packet, MAX_PACKET_CHAN_SIZE),
	}
}

// FlushPackets flushes all packets in the packet channel buffer.
func (conn *Connection) FlushPackets() {
	for {
		select {
		case packet := <-conn.packet:
			log.Warn("Flushing packet ", packet.Type)
		default:
			return
		}
	}
}

// NewConn creates a new UDP connection to the given IP and port.
//
// For multiple connection listening, use ConnManager.Listen() instead.
// This does not support multiple concurrent connections.
// Use this primarily for dialing.
//
// Can be used to dial or listen or both using the same methods (conn.Send() and conn.Receive()).
func NewConn(ip string, port int) (conn *Connection, err error) {
	conn = NewEmptyConnection()

	log.Info("Dialing ", ip, port)

	conn.RemoteAddr = &net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	}

	localAddr := &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0,
	}
	conn.conn, err = net.ListenUDP("udp", localAddr)
	if err != nil {
		return nil, err
	}

	conn.LocalAddr = conn.conn.LocalAddr().(*net.UDPAddr)

	go conn.receiver()

	log.Debug("Listening on: ", conn.LocalAddr)
	return conn, nil
}

// Close closes a connection.
func (conn *Connection) Close() error {
	if conn.conn != nil {
		close(conn.packet)
		return conn.conn.Close()
	}
	return nil
}

//------------------------------------------------------------------------------
//
// Packet sending / receiving / handling / helper functions
//
//------------------------------------------------------------------------------

// Send sends a packet to the Connection.RemoteAddr.
func (conn *Connection) Send(packet *Packet) error {
	buf := packet.Serialize()

	log.Debug("Sending from ", conn.LocalAddr, " to ", conn.RemoteAddr, " bytes: ", len(buf), " type: ", packet.Type, " data: ", string(packet.Data))

	n, err := conn.conn.WriteToUDP(buf, conn.RemoteAddr)
	if err != nil {
		log.Error("Error sending packet: ", err)
		return err
	}
	log.Debug("Sent ", n, " bytes to ", conn.RemoteAddr)

	return nil
}

// Receive waits for a packet to be received and returns it.
// If the packet is not the expected type, it will be logged and the next packet will be received.
// If the timeout is reached, it will return an error.
// Receive is blocking and will not return until a packet is received or the timeout is reached.
func (conn *Connection) Receive(expType MessageType, timeout time.Duration) (*Packet, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case packet := <-conn.packet:
			if packet.Type == expType {
				return packet, nil
			}
			log.Warn("Received unexpected packet ", packet.Type)
		case <-timer.C:
			return nil, fmt.Errorf("Timeout")
		}
	}
}

// Helper function to check if an error is a net.OpError with a closed error.
func errIsClosed(err error) bool {
	if err == nil {
		return false
	}
	if opErr, ok := err.(*net.OpError); ok {
		if opErr.Err == net.ErrClosed {
			return true
		}
	}
	return false
}

// receiver is the main loop for a connection.
// It reads from the connection and sends the packet/s to the packet channel.
func (conn *Connection) receiver() {
	buf := make([]byte, PACKET_SIZE)
	for {

		n, remoteAddr, err := conn.conn.ReadFromUDP(buf)
		conn.RemoteAddr = remoteAddr
		// If the connection was closed, stop the receiver
		if errIsClosed(err) {
			log.Info("Connection closed")
			return
		}
		if err != nil {
			log.Error("Error receiving from ", conn.RemoteAddr, "err: ", err)
			return
		}

		log.Debug("Received ", n, " bytes from ", conn.RemoteAddr)

		packet, err := Deserialize(buf[:n])
		if err != nil {
			log.Error("Error deserializing packet: ", err)
			return
		}

		log.Info("Received: ", string(packet.Data))

		log.Debug("Putting packet in channel: ", string(packet.Data))
		conn.packet <- packet

		packet, err = conn.handlePacket(packet)
		if err != nil {
			log.Error("Error handling packet: ", err)
			return
		}

		if packet != nil {
			err = conn.Send(packet)
			if err != nil {
				log.Error("Error sending packet: ", err)
				return
			}
		}

		select {
		case conn.packet <- packet:
		default:
			log.Warn("Packet queue full")
		}
	}
}

// handlePacket handles a packet received from the Connection.RemoteAddr.
func (conn *Connection) handlePacket(packet *Packet) (*Packet, error) {
	log.Debug("Received type: ", packet.Type)
	switch packet.Type {
	case MSG_CONN_PING:
		log.Info("Received PIGN!, responding with POGN!")
		return &Packet{
			Type: MSG_CONN_PONG,
			Data: []byte("POGN!"),
		}, nil
	case MSG_CONN_PONG:
		log.Info("Received POGN!")
		return nil, nil
	}
	return packet, nil
}
