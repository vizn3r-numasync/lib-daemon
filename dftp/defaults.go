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

	KEEP_ALIVE_MAX_RETRIES = 3 // PING retries
	MAX_RETRIES_TOP        = 5 // For critical
	MAX_RETRIES_BOT        = 3 // For bulk
)
