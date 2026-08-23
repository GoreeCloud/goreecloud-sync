package transfer

import "crypto/sha256"

const DefaultChunkSize = 1024 * 1024

type Chunk struct {
	Index int `json:"index"`
	Size int `json:"size"`
	Hash string `json:"hash"`
}

func Hash(data []byte) string {
	sum := sha256.Sum256(data)
	return string(sum[:])
}
