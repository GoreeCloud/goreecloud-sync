package transfer

import (
	"bytes"
	"strings"
	"testing"
)

const testTransferID = "00112233445566778899aabbccddeeff"

func TestPayloadOfferDecisionAndReceiptValidation(t *testing.T) {
	payload := []byte("authenticated GoreeCloud Sync payload")
	manifest, err := BuildManifest("example.txt", bytes.NewReader(payload), 8)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}
	offer := PayloadOffer{
		Version:    PayloadProtocolVersion,
		TransferID: testTransferID,
		Kind:       PayloadKindFile,
		Manifest:   manifest,
	}
	if err := offer.Validate(); err != nil {
		t.Fatalf("PayloadOffer.Validate() error = %v", err)
	}

	decision := PayloadDecision{Version: PayloadProtocolVersion, TransferID: testTransferID, Accepted: true}
	if err := decision.ValidateFor(offer); err != nil {
		t.Fatalf("PayloadDecision.ValidateFor() error = %v", err)
	}

	receipt := VerifiedPayloadReceipt(offer)
	if err := receipt.ValidateFor(offer); err != nil {
		t.Fatalf("PayloadReceipt.ValidateFor() error = %v", err)
	}
}

func TestPayloadOfferRejectsInvalidTransferIdentityAndKind(t *testing.T) {
	manifest, err := BuildManifest("text.txt", bytes.NewReader([]byte("hello")), DefaultChunkSize)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}

	tests := []PayloadOffer{
		{Version: PayloadProtocolVersion, TransferID: "short", Kind: PayloadKindText, Manifest: manifest},
		{Version: PayloadProtocolVersion, TransferID: strings.ToUpper(testTransferID), Kind: PayloadKindText, Manifest: manifest},
		{Version: PayloadProtocolVersion, TransferID: testTransferID, Kind: PayloadKind("unknown"), Manifest: manifest},
	}
	for _, offer := range tests {
		if err := offer.Validate(); err == nil {
			t.Fatalf("PayloadOffer.Validate() accepted invalid offer %+v", offer)
		}
	}
}

func TestPayloadDecisionAndReceiptAreBoundToOffer(t *testing.T) {
	manifest, err := BuildManifest("bound.bin", bytes.NewReader([]byte("payload")), DefaultChunkSize)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}
	offer := PayloadOffer{Version: PayloadProtocolVersion, TransferID: testTransferID, Kind: PayloadKindFile, Manifest: manifest}

	otherID := "ffeeddccbbaa99887766554433221100"
	if err := (PayloadDecision{Version: PayloadProtocolVersion, TransferID: otherID, Accepted: true}).ValidateFor(offer); err == nil {
		t.Fatal("PayloadDecision.ValidateFor() accepted another transfer ID")
	}

	receipt := VerifiedPayloadReceipt(offer)
	receipt.Hash = strings.Repeat("0", 64)
	if err := receipt.ValidateFor(offer); err == nil {
		t.Fatal("PayloadReceipt.ValidateFor() accepted another payload hash")
	}
}

func TestNewTransferIDProducesCanonicalRandomIdentifier(t *testing.T) {
	first, err := NewTransferID()
	if err != nil {
		t.Fatalf("NewTransferID() error = %v", err)
	}
	second, err := NewTransferID()
	if err != nil {
		t.Fatalf("NewTransferID() second error = %v", err)
	}
	if err := validateTransferID(first); err != nil {
		t.Fatalf("first transfer ID invalid: %v", err)
	}
	if err := validateTransferID(second); err != nil {
		t.Fatalf("second transfer ID invalid: %v", err)
	}
	if first == second {
		t.Fatal("NewTransferID() returned duplicate identifiers")
	}
}
