// dftp/handlers.go implements the DFTP handler and state machine with methods to handle different connection states.
// FUTURE: Implement ICE for DFTP

package dftp

type ConnState int

type ConnHandler struct {
	ConnState
}

const (
	// Default states
	CONN_STATE_CLOSED ConnState = iota // no connection with another node
	CONN_STATE_OPEN                    // connection is open with another node
	CONN_STATE_IDLE                    // connection is idle with another node, waiting for messages
	CONN_STATE_LISTEN                  // node is listening for incoming connections

	// Connection estabilishment states
	//// node is initiating
	CONN_ESTABLISH_INIT // sent SYN
	CONN_ESTABLISH_WAIT // received SYN-ACK, waaiting for ACK
	// when ACK received, switch to CONN_STATE_OPEN
	//// node is responding
	CONN_ESTABLISH_RESP // received SYN, sent SYN-ACK
	// after this, switch to CONN_STATE_OPEN

	// Keep alive states
	CONN_SENT_PING
	CONN_RECV_PONG

	// Abort states
	CONN_SENT_FIN
	CONN_RECV_FIN_ACK

	// Data transfer states
	CONN_TRANSFER_INIT
	CONN_TRANSFER_RECV_ACK // for when the node is sender
	CONN_TRANSFER_SENT_ACK // for when the node is receiver
	CONN_TRANSFER_SENT_DATA
)

func (h *ConnHandler) Handle(packet *Packet) (*Packet, error) {

	return packet, nil
}
