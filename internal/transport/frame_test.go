package transport

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestJSONFrameRoundTrip(t *testing.T) {
	original := Handshake{Protocol: Protocol, DeviceID: testDeviceID, Features: []string{"nearby", "sha256"}}
	var wire bytes.Buffer
	if err := WriteJSONFrame(&wire, original); err != nil {
		t.Fatalf("WriteJSONFrame() error = %v", err)
	}
	var decoded Handshake
	if err := ReadJSONFrame(&wire, &decoded); err != nil {
		t.Fatalf("ReadJSONFrame() error = %v", err)
	}
	if err := decoded.Validate(); err != nil {
		t.Fatalf("decoded handshake invalid: %v", err)
	}
	if decoded.DeviceID != original.DeviceID || len(decoded.Features) != 2 {
		t.Fatalf("decoded handshake = %#v", decoded)
	}
}

func TestReadJSONFrameRejectsOversizedLengthBeforeAllocation(t *testing.T) {
	var wire bytes.Buffer
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], MaxFrameSize+1)
	wire.Write(header[:])
	var decoded Handshake
	if err := ReadJSONFrame(&wire, &decoded); err == nil {
		t.Fatal("ReadJSONFrame() unexpectedly accepted oversized frame")
	}
}

func TestReadJSONFrameRejectsUnknownFields(t *testing.T) {
	payload := `{"protocol":"GC-SYNC/1","device_id":"11111111-1111-4111-8111-111111111111","features":[],"unexpected":true}`
	var wire bytes.Buffer
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	wire.Write(header[:])
	wire.WriteString(payload)
	var decoded Handshake
	if err := ReadJSONFrame(&wire, &decoded); err == nil {
		t.Fatal("ReadJSONFrame() unexpectedly accepted unknown field")
	}
}

func TestReadJSONFrameRejectsTruncatedPayload(t *testing.T) {
	var wire bytes.Buffer
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], 100)
	wire.Write(header[:])
	wire.WriteString("{}")
	var decoded Handshake
	if err := ReadJSONFrame(&wire, &decoded); err == nil {
		t.Fatal("ReadJSONFrame() unexpectedly accepted truncated frame")
	}
}

func TestWriteJSONFrameRejectsOversizedPayload(t *testing.T) {
	value := map[string]string{"payload": strings.Repeat("x", MaxFrameSize+1)}
	var wire bytes.Buffer
	if err := WriteJSONFrame(&wire, value); err == nil {
		t.Fatal("WriteJSONFrame() unexpectedly accepted oversized payload")
	}
	if wire.Len() != 0 {
		t.Fatalf("oversized frame wrote %d bytes", wire.Len())
	}
}
