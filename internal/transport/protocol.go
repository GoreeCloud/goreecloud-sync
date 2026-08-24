package transport

const (
	// Protocol identifies the GoreeCloud Sync transport protocol.
	Protocol = "GC-SYNC/1"
)

// Handshake represents the initial peer capability exchange.
type Handshake struct {
	Protocol string   `json:"protocol"`
	DeviceID string   `json:"device_id"`
	Features []string `json:"features"`
}
