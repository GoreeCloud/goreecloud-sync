package transport

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

const (
	// MaxFrameSize bounds one protocol frame before higher-level chunking is
	// applied. Large file payloads are transferred through the transfer engine,
	// not unbounded control messages.
	MaxFrameSize = 1 << 20 // 1 MiB
)

// WriteJSONFrame writes one length-prefixed JSON control frame.
func WriteJSONFrame(w io.Writer, value any) error {
	if w == nil {
		return fmt.Errorf("frame writer must not be nil")
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode frame: %w", err)
	}
	if len(payload) == 0 || len(payload) > MaxFrameSize {
		return fmt.Errorf("frame size %d is outside allowed bounds", len(payload))
	}

	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	if _, err := w.Write(header[:]); err != nil {
		return fmt.Errorf("write frame header: %w", err)
	}
	if _, err := w.Write(payload); err != nil {
		return fmt.Errorf("write frame payload: %w", err)
	}
	return nil
}

// ReadJSONFrame reads one bounded length-prefixed JSON control frame.
// Unknown JSON fields are rejected so peers fail closed when control contracts
// unexpectedly diverge.
func ReadJSONFrame(r io.Reader, destination any) error {
	if r == nil {
		return fmt.Errorf("frame reader must not be nil")
	}
	if destination == nil {
		return fmt.Errorf("frame destination must not be nil")
	}

	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return fmt.Errorf("read frame header: %w", err)
	}
	size := binary.BigEndian.Uint32(header[:])
	if size == 0 || size > MaxFrameSize {
		return fmt.Errorf("frame size %d is outside allowed bounds", size)
	}
	payload := make([]byte, int(size))
	if _, err := io.ReadFull(r, payload); err != nil {
		return fmt.Errorf("read frame payload: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode frame: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("frame contains trailing JSON value")
		}
		return fmt.Errorf("decode trailing frame data: %w", err)
	}
	return nil
}
