package transport

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/GoreeCloud/goreecloud-sync/internal/transfer"
)

type payloadReceiveResult struct {
	offer   transfer.PayloadOffer
	receipt transfer.PayloadReceipt
	payload []byte
	err     error
}

func TestAuthenticatedPayloadTransferRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		kind    transfer.PayloadKind
		payload []byte
	}{
		{
			name:    "file",
			kind:    transfer.PayloadKindFile,
			payload: bytes.Repeat([]byte("goreecloud-sync-file"), (2*transfer.DefaultChunkSize/len("goreecloud-sync-file"))+23),
		},
		{
			name:    "text",
			kind:    transfer.PayloadKindText,
			payload: []byte("private text transfer over an authenticated GoreeCloud Sync peer"),
		},
		{
			name:    "empty file",
			kind:    transfer.PayloadKindFile,
			payload: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := newSecurePeerPair(t)
			defer client.Close()
			defer server.Close()
			bindTestPayloadTrust(t, client, server)

			manifest, err := transfer.BuildManifest("payload.bin", bytes.NewReader(tt.payload), transfer.DefaultChunkSize)
			if err != nil {
				t.Fatalf("BuildManifest() error = %v", err)
			}
			offer := transfer.PayloadOffer{
				Version:    transfer.PayloadProtocolVersion,
				TransferID: "00112233445566778899aabbccddeeff",
				Kind:       tt.kind,
				Manifest:   manifest,
			}

			received := make(chan payloadReceiveResult, 1)
			go func() {
				var staged bytes.Buffer
				gotOffer, receipt, receiveErr := server.ReceiveTransferPayload(func(candidate transfer.PayloadOffer) error {
					if candidate.TransferID != offer.TransferID {
						return fmt.Errorf("unexpected transfer ID")
					}
					return nil
				}, &staged)
				received <- payloadReceiveResult{offer: gotOffer, receipt: receipt, payload: append([]byte(nil), staged.Bytes()...), err: receiveErr}
			}()

			receipt, err := client.SendTransferPayload(offer, bytes.NewReader(tt.payload))
			if err != nil {
				t.Fatalf("SendTransferPayload() error = %v", err)
			}
			if err := receipt.ValidateFor(offer); err != nil {
				t.Fatalf("sender receipt invalid: %v", err)
			}

			result := <-received
			if result.err != nil {
				t.Fatalf("ReceiveTransferPayload() error = %v", result.err)
			}
			if result.offer.TransferID != offer.TransferID {
				t.Fatalf("received transfer ID = %q, want %q", result.offer.TransferID, offer.TransferID)
			}
			if err := result.receipt.ValidateFor(offer); err != nil {
				t.Fatalf("receiver receipt invalid: %v", err)
			}
			if !bytes.Equal(result.payload, tt.payload) {
				t.Fatalf("received payload length = %d, want %d", len(result.payload), len(tt.payload))
			}
		})
	}
}

func TestAuthenticatedPayloadTransferRequiresExplicitReceiverAuthorization(t *testing.T) {
	client, server := newSecurePeerPair(t)
	defer client.Close()
	defer server.Close()
	bindTestPayloadTrust(t, client, server)

	payload := []byte("must not be sent before authorization")
	manifest, err := transfer.BuildManifest("denied.txt", bytes.NewReader(payload), transfer.DefaultChunkSize)
	if err != nil {
		t.Fatal(err)
	}
	offer := transfer.PayloadOffer{Version: transfer.PayloadProtocolVersion, TransferID: "00112233445566778899aabbccddeeff", Kind: transfer.PayloadKindText, Manifest: manifest}

	received := make(chan payloadReceiveResult, 1)
	go func() {
		var staged bytes.Buffer
		gotOffer, receipt, receiveErr := server.ReceiveTransferPayload(func(transfer.PayloadOffer) error {
			return fmt.Errorf("local policy denied transfer")
		}, &staged)
		received <- payloadReceiveResult{offer: gotOffer, receipt: receipt, payload: append([]byte(nil), staged.Bytes()...), err: receiveErr}
	}()

	if _, err := client.SendTransferPayload(offer, bytes.NewReader(payload)); !errors.Is(err, ErrPayloadTransferRejected) {
		t.Fatalf("SendTransferPayload() error = %v, want rejection", err)
	}
	result := <-received
	if !errors.Is(result.err, ErrPayloadTransferAuthorizationDenied) {
		t.Fatalf("ReceiveTransferPayload() error = %v, want authorization denial", result.err)
	}
	if len(result.payload) != 0 {
		t.Fatalf("receiver staged %d bytes before authorization", len(result.payload))
	}
	if client.IsClosed() || server.IsClosed() {
		t.Fatal("policy rejection should leave an otherwise synchronized secure peer reusable")
	}
}

func TestPayloadTransferRejectsSecurePeerWithoutDurableTrustBinding(t *testing.T) {
	client, server := newSecurePeerPair(t)
	defer server.Close()

	manifest, err := transfer.BuildManifest("unbound.bin", bytes.NewReader([]byte("data")), transfer.DefaultChunkSize)
	if err != nil {
		t.Fatal(err)
	}
	offer := transfer.PayloadOffer{Version: transfer.PayloadProtocolVersion, TransferID: "00112233445566778899aabbccddeeff", Kind: transfer.PayloadKindFile, Manifest: manifest}
	if _, err := client.SendTransferPayload(offer, bytes.NewReader([]byte("data"))); !errors.Is(err, ErrPayloadTransferPeerNotTrusted) {
		t.Fatalf("SendTransferPayload() error = %v, want trusted-peer requirement", err)
	}
	if !client.IsClosed() {
		t.Fatal("unbound secure peer was not closed fail-closed")
	}
}

func TestPayloadTransferClosesOnSourceIntegrityMismatch(t *testing.T) {
	client, server := newSecurePeerPair(t)
	defer client.Close()
	defer server.Close()
	bindTestPayloadTrust(t, client, server)

	original := bytes.Repeat([]byte("a"), transfer.DefaultChunkSize+19)
	manifest, err := transfer.BuildManifest("mismatch.bin", bytes.NewReader(original), transfer.DefaultChunkSize)
	if err != nil {
		t.Fatal(err)
	}
	offer := transfer.PayloadOffer{Version: transfer.PayloadProtocolVersion, TransferID: "00112233445566778899aabbccddeeff", Kind: transfer.PayloadKindFile, Manifest: manifest}
	corrupt := append([]byte(nil), original...)
	corrupt[0] = 'b'

	receiveErr := make(chan error, 1)
	go func() {
		var staged bytes.Buffer
		_, _, err := server.ReceiveTransferPayload(func(transfer.PayloadOffer) error { return nil }, &staged)
		receiveErr <- err
	}()

	if _, err := client.SendTransferPayload(offer, bytes.NewReader(corrupt)); err == nil || !strings.Contains(err.Error(), "verify source chunk") {
		t.Fatalf("SendTransferPayload() error = %v, want source integrity rejection", err)
	}
	if !client.IsClosed() {
		t.Fatal("sender peer remained open after stream integrity failure")
	}
	if err := <-receiveErr; err == nil {
		t.Fatal("receiver unexpectedly accepted transfer after sender integrity failure")
	}
}

func bindTestPayloadTrust(t *testing.T, client, server *PeerConn) {
	t.Helper()
	if err := client.BindAuthenticatedKeyFingerprint(strings.Repeat("a", 64)); err != nil {
		t.Fatalf("bind client trust: %v", err)
	}
	if err := server.BindAuthenticatedKeyFingerprint(strings.Repeat("b", 64)); err != nil {
		t.Fatalf("bind server trust: %v", err)
	}
}
