package transport

import (
	"fmt"
	"strings"
)

const (
	// Protocol identifies the GoreeCloud Sync transport protocol.
	Protocol = "GC-SYNC/1"

	// MaxHandshakeFeatures bounds peer-controlled capability metadata.
	MaxHandshakeFeatures = 32
)

// Handshake represents the initial peer capability exchange. This exchange is
// capability negotiation only; it does not establish trust or authorization.
type Handshake struct {
	Protocol string   `json:"protocol"`
	DeviceID string   `json:"device_id"`
	Features []string `json:"features"`
}

// Validate rejects malformed or unsupported peer handshakes before later
// pairing and authenticated-encryption layers consume them.
func (h Handshake) Validate() error {
	if h.Protocol != Protocol {
		return fmt.Errorf("unsupported sync protocol %q", h.Protocol)
	}
	if !validDeviceID(h.DeviceID) {
		return fmt.Errorf("device ID must be a canonical lowercase UUID")
	}
	if len(h.Features) > MaxHandshakeFeatures {
		return fmt.Errorf("too many handshake features")
	}
	seen := make(map[string]struct{}, len(h.Features))
	for _, feature := range h.Features {
		if !validFeature(feature) {
			return fmt.Errorf("invalid handshake feature %q", feature)
		}
		if _, exists := seen[feature]; exists {
			return fmt.Errorf("duplicate handshake feature %q", feature)
		}
		seen[feature] = struct{}{}
	}
	return nil
}

func validFeature(value string) bool {
	if value == "" || len(value) > 64 || strings.TrimSpace(value) != value {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func validDeviceID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, r := range value {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
				return false
			}
		}
	}
	return true
}
