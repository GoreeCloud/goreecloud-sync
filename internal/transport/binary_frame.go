package transport

import (
	"encoding/binary"
	"fmt"
	"io"
)

// WriteBinaryFrame writes one non-empty bounded binary payload frame using the
// same four-byte big-endian length prefix as the control-frame transport. Large
// payloads must be divided into transfer-engine chunks before reaching here.
func WriteBinaryFrame(w io.Writer, payload []byte) error {
	if w == nil {
		return fmt.Errorf("binary frame writer must not be nil")
	}
	if len(payload) == 0 || len(payload) > MaxFrameSize {
		return fmt.Errorf("binary frame size %d is outside allowed bounds", len(payload))
	}

	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	if _, err := w.Write(header[:]); err != nil {
		return fmt.Errorf("write binary frame header: %w", err)
	}
	if _, err := w.Write(payload); err != nil {
		return fmt.Errorf("write binary frame payload: %w", err)
	}
	return nil
}

// ReadBinaryFrame reads one binary frame and rejects an excessive declared size
// before allocating the payload buffer. maxSize is a caller-supplied semantic
// bound for the expected transfer chunk and may not exceed MaxFrameSize.
func ReadBinaryFrame(r io.Reader, maxSize int) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("binary frame reader must not be nil")
	}
	if maxSize <= 0 || maxSize > MaxFrameSize {
		return nil, fmt.Errorf("binary frame maximum %d is outside allowed bounds", maxSize)
	}

	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, fmt.Errorf("read binary frame header: %w", err)
	}
	size := binary.BigEndian.Uint32(header[:])
	if size == 0 || size > MaxFrameSize || size > uint32(maxSize) {
		return nil, fmt.Errorf("binary frame size %d is outside allowed bounds", size)
	}
	payload := make([]byte, int(size))
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, fmt.Errorf("read binary frame payload: %w", err)
	}
	return payload, nil
}
