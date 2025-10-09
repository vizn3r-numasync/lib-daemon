// dftp/defaults.go sets the default values for the protocol
// For now, the protocol is very simple, so the protocol uses static values

package dftp

// Default values for the protocol
const (
	HEADER_SIZE = 20   // bytes
	DATA_SIZE   = 1400 // bytes
	PACKET_SIZE = HEADER_SIZE + DATA_SIZE
	WINDOW_SIZE = 8 // bytes
	BITMAP_SIZE = 1 // bytes

	TIMEOUT            = 500       // ms
	HEARTBEAT_INTERVAL = 30 * 1000 // ms
	IDLE_TIMEOUT       = 60 * 1000 // ms

	KEEP_ALIVE_MAX_RETRIES = 3   // PING retries
	MAX_RETRIES_TOP        = 5   // For critical
	MAX_RETRIES_BOT        = 3   // For bulk
	MAX_PACKET_CHAN_SIZE   = 100 // How many packets to buffer
)

// Message types
// 0x00 - 0x0F: Connection Management
const (
	MSG_HELLO MessageType = 0x00 + iota
	MSG_PIGN
	MSG_POGN
	MSG_BYE
)

// 0x10 - 0x1F: Daemon <-> Server
const (
	MSG_REGISTER MessageType = 0x10 + iota
	MSG_PEER_INFO
)

// 0x20 - 0x2F: Daemon <->  Daemon
const (
	MSG_TRANSFER_INIT MessageType = 0x20 + iota
	MSG_DATA
	MSG_CHECK
	MSG_MISSING
	MSG_COMPLETE
)

// 0xF0 - 0xFF: Errors and debug
const (
	MSG_DEBUG MessageType = 0xF0 + iota
	MSG_ERROR
	MSG_ERR_NOT_FOUND
)

// Flags
const (
	FLAG_RESERVED Flags = 1 << iota // 0
	FLAG_PRIO                       // 1
	FLAG_MORE                       // 2
	FLAG_NACK                       // 3
	FLAG_RST                        // 4
	FLAG_FIN                        // 5
	FLAG_ACK                        // 6
	FLAG_SYN                        // 7
)
