package transfer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

const ManifestVersion = 1

// Manifest describes one file as a deterministic sequence of bounded chunks.
// It is a source-level Development contract and is not yet a Stable wire format.
type Manifest struct {
	Version   int     `json:"version"`
	Filename  string  `json:"filename"`
	Size      int64   `json:"size"`
	ChunkSize int     `json:"chunk_size"`
	Hash      string  `json:"hash"`
	Chunks    []Chunk `json:"chunks"`
}

// BuildManifest reads a payload once and records per-chunk and whole-payload
// SHA-256 integrity metadata. The current Development implementation limits
// chunks to DefaultChunkSize so memory use for one chunk remains bounded.
func BuildManifest(filename string, r io.Reader, chunkSize int) (Manifest, error) {
	if err := validateFilename(filename); err != nil {
		return Manifest{}, err
	}
	if r == nil {
		return Manifest{}, fmt.Errorf("manifest reader must not be nil")
	}
	if err := validateChunkSize(chunkSize); err != nil {
		return Manifest{}, err
	}

	wholeHash := sha256.New()
	buffer := make([]byte, chunkSize)
	chunks := make([]Chunk, 0)
	var total int64

	for index := 0; ; index++ {
		n, err := io.ReadFull(r, buffer)
		if n > 0 {
			data := buffer[:n]
			if _, hashErr := wholeHash.Write(data); hashErr != nil {
				return Manifest{}, fmt.Errorf("hash payload: %w", hashErr)
			}
			chunks = append(chunks, Chunk{
				Index: index,
				Size:  n,
				Hash:  Hash(data),
			})
			total += int64(n)
		}

		switch err {
		case nil:
			continue
		case io.EOF, io.ErrUnexpectedEOF:
			manifest := Manifest{
				Version:   ManifestVersion,
				Filename:  filename,
				Size:      total,
				ChunkSize: chunkSize,
				Hash:      hex.EncodeToString(wholeHash.Sum(nil)),
				Chunks:    chunks,
			}
			if validateErr := manifest.Validate(); validateErr != nil {
				return Manifest{}, fmt.Errorf("validate built manifest: %w", validateErr)
			}
			return manifest, nil
		default:
			return Manifest{}, fmt.Errorf("read payload for manifest: %w", err)
		}
	}
}

// Validate checks structural and integrity-metadata invariants without reading
// the underlying payload.
func (m Manifest) Validate() error {
	if m.Version != ManifestVersion {
		return fmt.Errorf("unsupported manifest version %d", m.Version)
	}
	if err := validateFilename(m.Filename); err != nil {
		return err
	}
	if m.Size < 0 {
		return fmt.Errorf("manifest size must not be negative")
	}
	if err := validateChunkSize(m.ChunkSize); err != nil {
		return err
	}
	if err := validateDigest(m.Hash); err != nil {
		return fmt.Errorf("manifest payload hash: %w", err)
	}

	if m.Size == 0 {
		if len(m.Chunks) != 0 {
			return fmt.Errorf("empty payload manifest must not contain chunks")
		}
		if m.Hash != Hash(nil) {
			return fmt.Errorf("empty payload manifest hash is invalid")
		}
		return nil
	}
	if len(m.Chunks) == 0 {
		return fmt.Errorf("non-empty payload manifest requires chunks")
	}

	expectedChunks := m.Size / int64(m.ChunkSize)
	if m.Size%int64(m.ChunkSize) != 0 {
		expectedChunks++
	}
	if int64(len(m.Chunks)) != expectedChunks {
		return fmt.Errorf("manifest chunk count %d does not match expected %d", len(m.Chunks), expectedChunks)
	}

	var total int64
	for index, chunk := range m.Chunks {
		if chunk.Index != index {
			return fmt.Errorf("manifest chunk index %d does not match position %d", chunk.Index, index)
		}
		if chunk.Size <= 0 || chunk.Size > m.ChunkSize {
			return fmt.Errorf("manifest chunk %d size %d is outside allowed bounds", index, chunk.Size)
		}
		if index < len(m.Chunks)-1 && chunk.Size != m.ChunkSize {
			return fmt.Errorf("manifest chunk %d must use the full configured chunk size", index)
		}
		if err := validateDigest(chunk.Hash); err != nil {
			return fmt.Errorf("manifest chunk %d hash: %w", index, err)
		}
		total += int64(chunk.Size)
	}
	if total != m.Size {
		return fmt.Errorf("manifest chunk bytes %d do not match payload size %d", total, m.Size)
	}
	return nil
}

// VerifyChunk checks one received chunk against the exact size and digest in
// the manifest entry.
func VerifyChunk(expected Chunk, data []byte) error {
	if expected.Index < 0 {
		return fmt.Errorf("chunk index must not be negative")
	}
	if expected.Size <= 0 || expected.Size > DefaultChunkSize {
		return fmt.Errorf("chunk size %d is outside allowed bounds", expected.Size)
	}
	if err := validateDigest(expected.Hash); err != nil {
		return fmt.Errorf("chunk hash: %w", err)
	}
	if len(data) != expected.Size {
		return fmt.Errorf("chunk %d size %d does not match expected %d", expected.Index, len(data), expected.Size)
	}
	if actual := Hash(data); actual != expected.Hash {
		return fmt.Errorf("chunk %d integrity verification failed", expected.Index)
	}
	return nil
}

// VerifyPayload streams a received payload through every manifest chunk and
// then verifies the whole-payload digest. Completion is not accepted when a
// chunk is truncated, corrupt, or followed by undeclared trailing bytes.
func VerifyPayload(manifest Manifest, r io.Reader) error {
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("validate manifest: %w", err)
	}
	if r == nil {
		return fmt.Errorf("payload reader must not be nil")
	}

	wholeHash := sha256.New()
	buffer := make([]byte, manifest.ChunkSize)
	for _, expected := range manifest.Chunks {
		data := buffer[:expected.Size]
		if _, err := io.ReadFull(r, data); err != nil {
			return fmt.Errorf("read chunk %d: %w", expected.Index, err)
		}
		if err := VerifyChunk(expected, data); err != nil {
			return err
		}
		if _, err := wholeHash.Write(data); err != nil {
			return fmt.Errorf("hash verified payload: %w", err)
		}
	}

	if n, err := io.CopyN(io.Discard, r, 1); n > 0 {
		return fmt.Errorf("payload contains undeclared trailing data")
	} else if err != nil && err != io.EOF {
		return fmt.Errorf("check payload boundary: %w", err)
	}

	if actual := hex.EncodeToString(wholeHash.Sum(nil)); actual != manifest.Hash {
		return fmt.Errorf("whole-payload integrity verification failed")
	}
	return nil
}

func validateFilename(filename string) error {
	if strings.TrimSpace(filename) == "" {
		return fmt.Errorf("manifest filename is required")
	}
	if strings.ContainsRune(filename, '\x00') {
		return fmt.Errorf("manifest filename contains a NUL byte")
	}
	if filename == "." || filename == ".." {
		return fmt.Errorf("manifest filename must be a path-independent leaf name")
	}
	if strings.ContainsAny(filename, `/\\`) {
		return fmt.Errorf("manifest filename must not contain path separators")
	}
	return nil
}

func validateChunkSize(chunkSize int) error {
	if chunkSize <= 0 || chunkSize > DefaultChunkSize {
		return fmt.Errorf("chunk size %d is outside current allowed range 1..%d", chunkSize, DefaultChunkSize)
	}
	return nil
}

func validateDigest(digest string) error {
	if len(digest) != sha256.Size*2 {
		return fmt.Errorf("SHA-256 digest must contain %d lowercase hexadecimal characters", sha256.Size*2)
	}
	if digest != strings.ToLower(digest) {
		return fmt.Errorf("SHA-256 digest must use lowercase hexadecimal")
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return fmt.Errorf("SHA-256 digest is not valid hexadecimal: %w", err)
	}
	return nil
}
