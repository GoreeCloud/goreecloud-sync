package transport

import (
	"bytes"
	"testing"
)

type boundedPartialWriter struct {
	buffer bytes.Buffer
	max    int
}

func (w *boundedPartialWriter) Write(p []byte) (int, error) {
	if len(p) > w.max {
		p = p[:w.max]
	}
	return w.buffer.Write(p)
}

type zeroProgressWriter struct{}

func (zeroProgressWriter) Write([]byte) (int, error) { return 0, nil }

func TestFrameWritersCompleteValidPartialWrites(t *testing.T) {
	jsonWriter := &boundedPartialWriter{max: 2}
	wantControl := Handshake{Protocol: Protocol, DeviceID: "11111111-1111-1111-1111-111111111111", Features: []string{"payload-transfer"}}
	if err := WriteJSONFrame(jsonWriter, wantControl); err != nil {
		t.Fatalf("WriteJSONFrame() partial writer error = %v", err)
	}
	var gotControl Handshake
	if err := ReadJSONFrame(bytes.NewReader(jsonWriter.buffer.Bytes()), &gotControl); err != nil {
		t.Fatalf("ReadJSONFrame() after partial writes error = %v", err)
	}
	if gotControl.DeviceID != wantControl.DeviceID || gotControl.Protocol != wantControl.Protocol {
		t.Fatalf("control round trip = %+v, want %+v", gotControl, wantControl)
	}

	binaryWriter := &boundedPartialWriter{max: 3}
	wantPayload := []byte("bounded payload")
	if err := WriteBinaryFrame(binaryWriter, wantPayload); err != nil {
		t.Fatalf("WriteBinaryFrame() partial writer error = %v", err)
	}
	gotPayload, err := ReadBinaryFrame(bytes.NewReader(binaryWriter.buffer.Bytes()), len(wantPayload))
	if err != nil {
		t.Fatalf("ReadBinaryFrame() after partial writes error = %v", err)
	}
	if !bytes.Equal(gotPayload, wantPayload) {
		t.Fatalf("binary payload = %q, want %q", gotPayload, wantPayload)
	}
}

func TestFrameWritersRejectZeroProgress(t *testing.T) {
	if err := WriteJSONFrame(zeroProgressWriter{}, struct{ Value string `json:"value"` }{Value: "test"}); err == nil {
		t.Fatal("WriteJSONFrame() accepted a zero-progress writer")
	}
	if err := WriteBinaryFrame(zeroProgressWriter{}, []byte("test")); err == nil {
		t.Fatal("WriteBinaryFrame() accepted a zero-progress writer")
	}
}
