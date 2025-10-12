// dftp/connection.go handles the connection between two DFTP nodes.
// It handles packet sending and receiving, and the packet channel.

package dftp

import (
	"context"
	"fmt"
	"hash/fnv"
	"net"
	"sync"

	"github.com/gofrs/uuid"
)

type ConnState int

type Connection struct {
	SessionID  uint32
	ConnID     uint8
	LocalAddr  *net.UDPAddr
	RemoteAddr *net.UDPAddr
	State      ConnState

	conn     *net.UDPConn
	packet   chan *Packet
	chunkMap map[uint32]*Chunk

	ready  chan struct{}
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex
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
		State:      STATE_CLOSED,
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

var usedSessionIDs = sync.Map{}

// NewConn creates a new UDP connection to the given IP and port.
//
// For multiple connection listening, use ConnManager.Listen() instead.
// This does not support multiple concurrent connections.
// Use this primarily for dialing.
//
// Can be used to dial or listen or both using the same methods (conn.Send() and conn.Receive()).
func NewConn(ip string, port int, sessionID uint32) (conn *Connection, err error) {
	conn = NewEmptyConnection()

	conn.RemoteAddr = &net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	}

	localAddr := &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0,
	}

	// This will be handled by server in the future
	if sessionID == 0 {
		for {
			uuid := uuid.Must(uuid.NewV4())
			h := fnv.New32a()
			h.Write(uuid[:])
			if _, ok := usedSessionIDs.Load(h.Sum32()); ok {
				continue
			}
			conn.SessionID = h.Sum32()
			usedSessionIDs.Store(conn.SessionID, true)
			break
		}
	} else {
		conn.SessionID = sessionID
	}

	log.Info("Dialing ", ip, ":", port, " sessionID: ", conn.SessionID)

	conn.conn, err = net.ListenUDP("udp", localAddr)
	if err != nil {
		return nil, err
	}

	conn.LocalAddr = conn.conn.LocalAddr().(*net.UDPAddr)

	go conn.receiver()

	// Make sure receiver() started first
	<-conn.ready

	conn.State = STATE_IDLE

	log.Debug("Listening on: ", conn.LocalAddr)
	return conn, nil
}

// Close closes a connection.
func (conn *Connection) CloseWithoutConn() error {
	log.Debug("Closing connection, sessionID: ", conn.SessionID)
	conn.mu.Lock()
	conn.State = STATE_CLOSED
	conn.cancel()
	log.Debug("Waiting for receiver to exit...")
	conn.wg.Wait()
	log.Debug("Receiver exited, closing channel")
	close(conn.packet)
	log.Debug("Connection closed")
	conn.mu.Unlock()
	return nil
}

// Close closes a connection.
func (conn *Connection) Close() error {
	log.Debug("Closing connection, sessionID: ", conn.SessionID)
	conn.mu.Lock()
	conn.State = STATE_CLOSED
	conn.cancel()
	if conn.conn != nil {
		err := conn.conn.Close()
		if err != nil {
			conn.mu.Unlock()
			log.Error("Error closing connection: ", err)
			return err
		}
	}
	log.Debug("Waiting for receiver to exit...")
	conn.wg.Wait()
	log.Debug("Receiver exited, closing channel")
	close(conn.packet)
	log.Debug("Connection closed")
	conn.mu.Unlock()
	return nil
}

//------------------------------------------------------------------------------
//
// Packet sending / receiving / handling / helper functions
//
//------------------------------------------------------------------------------

// Send sends a packet to the Connection.RemoteAddr.
func (conn *Connection) Send(packet *Packet) error {
	p := *packet

	p.SessionID = conn.SessionID

	buf := p.Serialize()

	log.Debug("Sending from ", conn.LocalAddr, " to ", conn.RemoteAddr, " bytes: ", len(buf), " type: ", packet.Type, " data: ", string(packet.Data))

	conn.mu.RLock()
	addr := conn.RemoteAddr
	conn.mu.RUnlock()

	n, err := conn.conn.WriteToUDP(buf, addr)
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

var connBufferPool = sync.Pool{
	New: func() any {
		return make([]byte, PACKET_SIZE)
	},
}

// receiver is the main loop for a connection.
// It reads from the connection and sends the packet/s to the packet channel.
func (conn *Connection) receiver() {
	conn.wg.Add(1)
	defer conn.wg.Done()

	close(conn.ready)

	for {
		select {
		case <-conn.ctx.Done():
			log.Info("Connection closed")
			return
		default:
		}

		bufPtr := connBufferPool.Get().([]byte)

		n, remoteAddr, err := conn.conn.ReadFromUDP(bufPtr)
		if err != nil {
			connBufferPool.Put(bufPtr)
			if errIsClosed(err) || conn.State == STATE_CLOSED {
				log.Info("Connection closed")
				return
			}

			if err, ok := err.(*net.OpError); ok && err.Timeout() {
				continue
			}

			log.Error("Error receiving from ", conn.RemoteAddr, "err: ", err)
			return
		}
		if conn.ctx.Err() != nil {
			log.Info("Connection closed")
			return
		}

		if conn.RemoteAddr == nil {
			conn.mu.Lock()
			conn.RemoteAddr = remoteAddr
			conn.mu.Unlock()
		}

		log.Debug("Received ", n, " bytes from ", conn.RemoteAddr)

		packet, err := Deserialize(bufPtr[:n])
		connBufferPool.Put(bufPtr)
		if err != nil {
			log.Error("Error deserializing packet: ", err)
			return
		}

		select {
		case <-conn.ctx.Done():
			log.Info("Connection closed while receiving packet")
			return
		case conn.packet <- packet:
			log.Debug("Putting packet in channel: ", string(packet.Data))
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
