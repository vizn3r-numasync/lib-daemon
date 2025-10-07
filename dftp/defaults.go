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
	MSG_CONN_SYN       MessageType = 0x00 + iota // 0x00
	MSG_CONN_SYN_ACK                             // 0x01
	MSG_CONN_PING                                // 0x02
	MSG_CONN_PONG                                // 0x03
	MSG_CONN_CLOSE                               // 0x04
	MSG_CONN_CLOSE_ACK                           // 0x05
)

// 0x10 - 0x1F: Daemon <-> Server
const (
	MSG_DAEMON_REGISTER          MessageType = 0x10 + iota // 0x10
	MSG_DAEMON_REGISTER_ACK                                // 0x11
	MSG_DAEMON_HEARTBEAT                                   // 0x12
	MSG_DAEMON_HEARTBEAT_ACK                               // 0x13
	MSG_FILE_METADATA_UPLOAD                               // 0x14
	MSG_FILE_METADATA_UPLOAD_ACK                           // 0x15
	MSG_STORAGE_ASSIGN_REQUEST                             // 0x16
	MSG_STORAGE_ASSIGN_RESPONSE                            // 0x17
	MSG_P2P_COORD_REQUEST                                  // 0x18
	MSG_P2P_COORD_RESPONSE                                 // 0x19
)

// 0x20 - 0x2F: Daemon <->  Daemon
const (
	MSG_TRANSFER_INIT         MessageType = 0x20 + iota // 0x20
	MSG_TRANSFER_INIT_ACK                               // 0x21
	MSG_TRANSFER_COMPLETE                               // 0x22
	MSG_TRANSFER_COMPLETE_ACK                           // 0x23
	MSG_CHUNK_DATA                                      // 0x24
	MSG_CHUNK_DATA_ACK                                  // 0x25
	MSG_CHUNK_REQUEST                                   // 0x26
	MSG_CHUNK_REQUEST_NACK                              // 0x27
)

// 0xF0 - 0xFF: Errors and debug
const (
	MSG_DEBUG_MESSAGE          MessageType = 0xF0 + iota // 0xF0
	MSG_ERROR_GENERIC                                    // 0xF1
	MSG_ERROR_AUTH                                       // 0xF2
	MSG_ERROR_NOT_FOUND                                  // 0xF3
	MSG_ERROR_QUOTA_EXCEEDED                             // 0xF4
	MSG_ERROR_RATE_LIMITED                               // 0xF5
	MSG_ERROR_INVALID_SESSION                            // 0xF6
	MSG_ERROR_VERSION_MISMATCH                           // 0xF7
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
