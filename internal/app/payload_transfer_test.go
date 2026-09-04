package app

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/GoreeCloud/goreecloud-sync/internal/transfer"
)

type factoryPayloadResult struct {
	payload []byte
	err     error
}

func TestSecurePeerFactoryPayloadTransferWithCurrentTrust(t *testing.T) {
	client, server, clientPeer, serverPeer := establishSecureFactoryPeers(t)
	defer clientPeer.Close()
	defer serverPeer.Close()

	payload := bytes.Repeat([]byte("trusted-transfer"), (transfer.DefaultChunkSize/len("trusted-transfer"))+71)
	manifest, err := transfer.BuildManifest("trusted.bin", bytes.NewReader(payload), transfer.DefaultChunkSize)
	if err != nil {
		t.Fatal(err)
	}
	offer := transfer.PayloadOffer{
		Version:    transfer.PayloadProtocolVersion,
		TransferID: "00112233445566778899aabbccddeeff",
		Kind:       transfer.PayloadKindFile,
		Manifest:   manifest,
	}

	received := make(chan factoryPayloadResult, 1)
	go func() {
		var staged bytes.Buffer
		_, _, receiveErr := server.factory.ReceiveTransferPayload(serverPeer, func(transfer.PayloadOffer) error { return nil }, &staged)
		received <- factoryPayloadResult{payload: append([]byte(nil), staged.Bytes()...), err: receiveErr}
	}()

	receipt, err := client.factory.SendTransferPayload(clientPeer, offer, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("SendTransferPayload() error = %v", err)
	}
	if err := receipt.ValidateFor(offer); err != nil {
		t.Fatalf("receipt invalid: %v", err)
	}
	result := <-received
	if result.err != nil {
		t.Fatalf("ReceiveTransferPayload() error = %v", result.err)
	}
	if !bytes.Equal(result.payload, payload) {
		t.Fatalf("received %d bytes, want %d", len(result.payload), len(payload))
	}
}

func TestSecurePeerFactoryBlocksPayloadTransferAfterRevocation(t *testing.T) {
	client, _, clientPeer, serverPeer := establishSecureFactoryPeers(t)
	defer serverPeer.Close()

	payload := []byte("blocked")
	manifest, err := transfer.BuildManifest("blocked.bin", bytes.NewReader(payload), transfer.DefaultChunkSize)
	if err != nil {
		t.Fatal(err)
	}
	offer := transfer.PayloadOffer{Version: transfer.PayloadProtocolVersion, TransferID: "00112233445566778899aabbccddeeff", Kind: transfer.PayloadKindFile, Manifest: manifest}
	if _, err := client.trust.Revoke(secureFactoryAccountID, secureFactoryServerID, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := client.factory.SendTransferPayload(clientPeer, offer, bytes.NewReader(payload)); !errors.Is(err, ErrSecurePeerTrustNotCurrent) {
		t.Fatalf("SendTransferPayload() error = %v, want current-trust failure", err)
	}
	if !clientPeer.IsClosed() {
		t.Fatal("revoked sender peer must close before transfer starts")
	}
}

func TestSecurePeerFactoryStopsPayloadBetweenChunksAfterRevocation(t *testing.T) {
	client, server, clientPeer, serverPeer := establishSecureFactoryPeers(t)
	defer clientPeer.Close()
	defer serverPeer.Close()

	payload := bytes.Repeat([]byte("x"), transfer.DefaultChunkSize+64)
	manifest, err := transfer.BuildManifest("revoked.bin", bytes.NewReader(payload), transfer.DefaultChunkSize)
	if err != nil {
		t.Fatal(err)
	}
	offer := transfer.PayloadOffer{Version: transfer.PayloadProtocolVersion, TransferID: "00112233445566778899aabbccddeeff", Kind: transfer.PayloadKindFile, Manifest: manifest}

	received := make(chan factoryPayloadResult, 1)
	go func() {
		var staged bytes.Buffer
		_, _, receiveErr := server.factory.ReceiveTransferPayload(serverPeer, func(transfer.PayloadOffer) error { return nil }, &staged)
		received <- factoryPayloadResult{payload: append([]byte(nil), staged.Bytes()...), err: receiveErr}
	}()

	source := &revokeAfterFirstRead{
		reader: bytes.NewReader(payload),
		revoke: func() error {
			_, revokeErr := client.trust.Revoke(secureFactoryAccountID, secureFactoryServerID, time.Now())
			return revokeErr
		},
	}
	if _, err := client.factory.SendTransferPayload(clientPeer, offer, source); !errors.Is(err, ErrSecurePeerTrustNotCurrent) {
		t.Fatalf("SendTransferPayload() error = %v, want current-trust failure", err)
	}
	result := <-received
	if result.err == nil {
		t.Fatal("receiver unexpectedly completed transfer after sender trust revocation")
	}
	if len(result.payload) != transfer.DefaultChunkSize {
		t.Fatalf("receiver staged %d bytes, want exactly one verified chunk", len(result.payload))
	}
}

type revokeAfterFirstRead struct {
	reader io.Reader
	revoke func() error
	once   sync.Once
	err    error
}

func (r *revokeAfterFirstRead) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.once.Do(func() { r.err = r.revoke() })
		if r.err != nil {
			return n, r.err
		}
	}
	return n, err
}
