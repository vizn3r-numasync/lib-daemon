// dftp/connection.go handles the connection between two DFTP nodes.
// It handles packet sending and receiving, and the packet channel.

package dftp

import (
	"fmt"
	"net"
	"time"
)

type ConnState int

type Connection struct {
	ConnState
	LocalSEQNum  uint32
	RemoteSEQNum uint32
	SessionID    uint32
	LocalAddr    *net.UDPAddr
	RemoteAddr   *net.UDPAddr

	conn   *net.UDPConn
	packet chan *Packet
}

const (
	STATE_CLOSED    ConnState = iota // Connection is closed
	STATE_IDLE                       // Waitig for peer connection
	STATE_CONNECTED                  // Connection established
	STATE_DATA                       // Data transfer in progress
)

//------------------------------------------------------------------------------
//
// Connection creation / destruction
//
//------------------------------------------------------------------------------

// NewEmptyConnection creates a new empty Connection struct.
// Use NewConn() to create a connection.
func NewEmptyConnection() *Connection {
	HandlersInit()
	return &Connection{
		ConnState:    STATE_CLOSED,
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

	conn.ConnState = STATE_IDLE

	log.Debug("Listening on: ", conn.LocalAddr)
	return conn, nil
}

// Close closes a connection.
func (conn *Connection) Close() error {
	conn.ConnState = STATE_CLOSED
	close(conn.packet)

	if conn.conn != nil {
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
		if errIsClosed(err) || conn.ConnState == STATE_CLOSED {
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

		select {
		case conn.packet <- packet:
			log.Debug("Putting packet in channel: ", string(packet.Data))
		default:
			log.Warn("Packet queue full")
		}

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

	}
}

// handlePacket handles a packet received from the Connection.RemoteAddr.
func (conn *Connection) handlePacket(packet *Packet) (*Packet, error) {
	m.RLock()
	defer m.RUnlock()
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
	packet, err := handler(packet)
	if err != nil {
		log.Error("Error handling packet: ", err)
		return &Packet{
			Type: MSG_ERROR,
			Data: []byte(err.Error()),
		}, err
	}
	return packet, nil
}
