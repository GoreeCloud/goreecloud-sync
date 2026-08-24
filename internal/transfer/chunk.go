package transfer

import (
	"crypto/sha256"
	"encoding/hex"
)

const DefaultChunkSize = 1024 * 1024

type Chunk struct {
	Index int    `json:"index"`
	Size  int    `json:"size"`
	Hash  string `json:"hash"`
}

func Hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
