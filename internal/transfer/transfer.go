package transfer

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

// ChunkSize defines the default transfer block size.
const ChunkSize = 1024 * 1024

// HashReader calculates a SHA-256 digest for transferred data.
func HashReader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
