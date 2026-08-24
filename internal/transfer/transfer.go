package transfer

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

// HashReader calculates a SHA-256 digest for transferred data.
func HashReader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
