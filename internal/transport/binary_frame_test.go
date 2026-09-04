package transport

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestBinaryFrameRoundTrip(t *testing.T) {
	payload := []byte("bounded encrypted payload chunk")
	var stream bytes.Buffer
	if err := WriteBinaryFrame(&stream, payload); err != nil {
		t.Fatalf("WriteBinaryFrame() error = %v", err)
	}
	got, err := ReadBinaryFrame(&stream, len(payload))
	if err != nil {
		t.Fatalf("ReadBinaryFrame() error = %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload = %q, want %q", got, payload)
	}
}

func TestReadBinaryFrameRejectsDeclaredSizeAboveSemanticBoundBeforePayload(t *testing.T) {
	var stream bytes.Buffer
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], 128)
	stream.Write(header[:])
	if _, err := ReadBinaryFrame(&stream, 64); err == nil {
		t.Fatal("ReadBinaryFrame() accepted declared size above semantic bound")
	}
}

func TestReadBinaryFrameRejectsGlobalOversizeBeforePayload(t *testing.T) {
	var stream bytes.Buffer
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(MaxFrameSize+1))
	stream.Write(header[:])
	if _, err := ReadBinaryFrame(&stream, MaxFrameSize); err == nil {
		t.Fatal("ReadBinaryFrame() accepted frame above global bound")
	}
}

func TestBinaryFrameRejectsEmptyAndInvalidBounds(t *testing.T) {
	if err := WriteBinaryFrame(&bytes.Buffer{}, nil); err == nil {
		t.Fatal("WriteBinaryFrame() accepted an empty payload")
	}
	if _, err := ReadBinaryFrame(bytes.NewReader(nil), 0); err == nil {
		t.Fatal("ReadBinaryFrame() accepted a zero maximum")
	}
	if _, err := ReadBinaryFrame(bytes.NewReader(nil), MaxFrameSize+1); err == nil {
		t.Fatal("ReadBinaryFrame() accepted maximum above global bound")
	}
}
