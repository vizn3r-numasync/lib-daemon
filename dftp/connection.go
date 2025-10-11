// dftp/connection.go handles the connection between two DFTP nodes.
// It handles packet sending and receiving, and the packet channel.

package dftp

import (
	"context"
	"fmt"
	"net"
	"sync"
)

type ConnState int

type Connection struct {
	ConnState
	ConnID     uint8
	LocalAddr  *net.UDPAddr
	RemoteAddr *net.UDPAddr

	// Chunk management
	chunkMap map[uint32]Chunk

	conn   *net.UDPConn
	packet chan *Packet

	ready chan struct{}

	ctx    context.Context
	cancel context.CancelFunc
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
	ctx, cancel := context.WithCancel(context.Background())
	return &Connection{
		ConnState:  STATE_CLOSED,
		LocalAddr:  nil,
		RemoteAddr: nil,
		conn:       nil,
		packet:     make(chan *Packet, MAX_PACKET_CHAN_SIZE),
		ready:      make(chan struct{}),
		ctx:        ctx,
		cancel:     cancel,
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

	// Make sure receiver() started first
	<-conn.ready

	conn.ConnState = STATE_IDLE

	log.Debug("Listening on: ", conn.LocalAddr)
	return conn, nil
}

// Close closes a connection.
func (conn *Connection) Close() error {
	conn.ConnState = STATE_CLOSED
	conn.cancel()
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

func (conn *Connection) Recv() (*Packet, error) {
	packet, ok := <-conn.packet
	if !ok {
		return nil, fmt.Errorf("Connection closed")
	}
	return packet, nil
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

var bufferPool = sync.Pool{
	New: func() any {
		b := make([]byte, PACKET_SIZE)
		return &b
	},
}

// receiver is the main loop for a connection.
// It reads from the connection and sends the packet/s to the packet channel.
func (conn *Connection) receiver() {
	bufPtr := bufferPool.Get().(*[]byte)
	buf := *bufPtr
	defer bufferPool.Put(bufPtr)

	close(conn.ready)

	for {
		select {
		case <-conn.ctx.Done():
			log.Info("Connection closed")
			return
		default:
		}

		n, remoteAddr, err := conn.conn.ReadFromUDP(buf)
		conn.RemoteAddr = remoteAddr

		if err != nil {
			if conn.ctx.Err() != nil {
				log.Info("Connection closed")
				return

			}
			if err, ok := err.(*net.OpError); ok && err.Timeout() {
				continue
			}

			if errIsClosed(err) || conn.ConnState == STATE_CLOSED {
				log.Info("Connection closed")
				return
			}

			log.Error("Error receiving from ", conn.RemoteAddr, "err: ", err)
			return
		}

		log.Debug("Received ", n, " bytes from ", conn.RemoteAddr)

		packet, err := Deserialize(buf[:n])
		if err != nil {
			log.Error("Error deserializing packet: ", err)
			return
		}

		select {
		case conn.packet <- packet:
			log.Debug("Putting packet in channel: ", string(packet.Data))
		case <-conn.ctx.Done():
			log.Info("Connection closed while receiving packet")
			return
		default:
			log.Warn("Packet queue full")
		}

		log.Info("Received: ", string(packet.Data))

		npacket, err := conn.handlePacket(packet)
		if err != nil {
			log.Error("Error handling packet: ", err)
			return
		}

		if npacket != nil {
			err = conn.Send(npacket)
			if err != nil {
				log.Error("Error sending packet: ", err)
				return
			}
		}
	}
}
