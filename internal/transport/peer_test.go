package transport

import (
	"net"
	"testing"
)

func TestPeerHandshakeRoundTrip(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close(); defer right.Close()
	sender, _ := AcceptPeer(left)
	receiver, _ := AcceptPeer(right)
	want := Handshake{Protocol: Protocol, DeviceID: "11111111-1111-1111-1111-111111111111", Features: []string{"resume"}}
	errCh := make(chan error, 1)
	go func() { errCh <- sender.SendHandshake(want) }()
	got, err := receiver.ReceiveHandshake()
	if err != nil { t.Fatal(err) }
	if err := <-errCh; err != nil { t.Fatal(err) }
	if got.DeviceID != want.DeviceID || got.Protocol != Protocol { t.Fatalf("unexpected handshake: %#v", got) }
}

func TestReceiveHandshakeRejectsInvalidProtocol(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close(); defer right.Close()
	receiver, _ := AcceptPeer(right)
	go func() { _ = WriteJSONFrame(left, Handshake{Protocol: "GC-SYNC/999", DeviceID: "11111111-1111-1111-1111-111111111111"}) }()
	if _, err := receiver.ReceiveHandshake(); err == nil { t.Fatal("expected invalid protocol error") }
}
