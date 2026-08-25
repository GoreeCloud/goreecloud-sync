package transport

import "testing"

const testDeviceID = "11111111-1111-4111-8111-111111111111"

func TestHandshakeValidateAcceptsBoundedCapabilities(t *testing.T) {
	h := Handshake{
		Protocol: Protocol,
		DeviceID: testDeviceID,
		Features: []string{"nearby", "resume-v1", "sha256"},
	}
	if err := h.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestHandshakeValidateFailsClosed(t *testing.T) {
	tests := []Handshake{
		{Protocol: "GC-SYNC/999", DeviceID: testDeviceID},
		{Protocol: Protocol, DeviceID: "../device"},
		{Protocol: Protocol, DeviceID: testDeviceID, Features: []string{"bad feature"}},
		{Protocol: Protocol, DeviceID: testDeviceID, Features: []string{"nearby", "nearby"}},
	}
	for i, handshake := range tests {
		if err := handshake.Validate(); err == nil {
			t.Fatalf("case %d unexpectedly validated", i)
		}
	}
}

func TestHandshakeValidateRejectsFeatureFlood(t *testing.T) {
	features := make([]string, MaxHandshakeFeatures+1)
	for i := range features {
		features[i] = "x"
	}
	h := Handshake{Protocol: Protocol, DeviceID: testDeviceID, Features: features}
	if err := h.Validate(); err == nil {
		t.Fatal("Validate() unexpectedly accepted excessive features")
	}
}
