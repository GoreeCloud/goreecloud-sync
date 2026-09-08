package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoreeCloud/goreecloud-sync/internal/transfer"
)

func publicationFixture(t *testing.T, payload string) (transfer.PayloadOffer, transfer.PayloadReceipt) {
	t.Helper()
	manifest, err := transfer.BuildManifest("report.txt", strings.NewReader(payload), 4)
	if err != nil {
		t.Fatal(err)
	}
	offer := transfer.PayloadOffer{
		Version:    transfer.PayloadProtocolVersion,
		TransferID: "00112233445566778899aabbccddeeff",
		Kind:       transfer.PayloadKindFile,
		Manifest:   manifest,
	}
	if err := offer.Validate(); err != nil {
		t.Fatal(err)
	}
	return offer, transfer.VerifiedPayloadReceipt(offer)
}

func TestLocalPayloadPublisherPublishesOnlyAfterAuthorizationAndStagedVerification(t *testing.T) {
	root := t.TempDir()
	offer, receipt := publicationFixture(t, "verified payload")
	var request PayloadPublicationRequest
	publisher, err := NewLocalPayloadPublisher("downloads", root, func(value PayloadPublicationRequest) error {
		request = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	published, err := publisher.Publish(offer, receipt, strings.NewReader("verified payload"))
	if err != nil {
		t.Fatal(err)
	}
	if request.DestinationID != "downloads" || request.TransferID != offer.TransferID || request.Filename != "report.txt" || request.Hash != offer.Manifest.Hash {
		t.Fatalf("unexpected authorization request: %+v", request)
	}
	if published.DestinationID != "downloads" || published.TransferID != offer.TransferID || published.Filename != "report.txt" || published.Hash != offer.Manifest.Hash {
		t.Fatalf("unexpected publication result: %+v", published)
	}
	content, err := os.ReadFile(filepath.Join(root, "report.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "verified payload" {
		t.Fatalf("published content=%q", content)
	}
	info, err := os.Stat(filepath.Join(root, "report.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("published mode=%#o grants group/other access", info.Mode().Perm())
	}
}

func TestLocalPayloadPublisherRejectsReceiptMismatchBeforeAuthorization(t *testing.T) {
	root := t.TempDir()
	offer, receipt := publicationFixture(t, "verified payload")
	receipt.Hash = strings.Repeat("0", 64)
	authorizeCalls := 0
	publisher, err := NewLocalPayloadPublisher("downloads", root, func(PayloadPublicationRequest) error {
		authorizeCalls++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := publisher.Publish(offer, receipt, strings.NewReader("verified payload")); err == nil {
		t.Fatal("mismatched receipt was accepted")
	}
	if authorizeCalls != 0 {
		t.Fatalf("authorizer called %d times for invalid receipt", authorizeCalls)
	}
	if _, err := os.Stat(filepath.Join(root, "report.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unexpected destination state: %v", err)
	}
}

func TestLocalPayloadPublisherDenialCreatesNoDestination(t *testing.T) {
	root := t.TempDir()
	offer, receipt := publicationFixture(t, "verified payload")
	publisher, err := NewLocalPayloadPublisher("downloads", root, func(PayloadPublicationRequest) error {
		return errors.New("policy denied")
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := publisher.Publish(offer, receipt, strings.NewReader("verified payload")); !errors.Is(err, ErrPayloadPublicationAuthorizationDenied) {
		t.Fatalf("Publish() err=%v want authorization denied", err)
	}
	if _, err := os.Stat(filepath.Join(root, "report.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unexpected destination state: %v", err)
	}
}

func TestLocalPayloadPublisherRejectsChangedStagingAfterVerifiedReceipt(t *testing.T) {
	root := t.TempDir()
	offer, receipt := publicationFixture(t, "verified payload")
	publisher, err := NewLocalPayloadPublisher("downloads", root, func(PayloadPublicationRequest) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := publisher.Publish(offer, receipt, strings.NewReader("tampered payload")); err == nil {
		t.Fatal("changed staged content was published")
	}
	if _, err := os.Stat(filepath.Join(root, "report.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unexpected destination state: %v", err)
	}
}

func TestLocalPayloadPublisherDoesNotOverwriteExistingDestination(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "report.txt")
	if err := os.WriteFile(target, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	offer, receipt := publicationFixture(t, "replacement")
	publisher, err := NewLocalPayloadPublisher("downloads", root, func(PayloadPublicationRequest) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := publisher.Publish(offer, receipt, strings.NewReader("replacement")); !errors.Is(err, ErrPayloadDestinationExists) {
		t.Fatalf("Publish() err=%v want destination exists", err)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "existing" {
		t.Fatalf("existing destination was modified: %q", content)
	}
}

func TestNewLocalPayloadPublisherRejectsMissingAuthorizationAndInvalidDestination(t *testing.T) {
	root := t.TempDir()
	if _, err := NewLocalPayloadPublisher("downloads", root, nil); err == nil {
		t.Fatal("nil authorizer was accepted")
	}
	if _, err := NewLocalPayloadPublisher("", root, func(PayloadPublicationRequest) error { return nil }); err == nil {
		t.Fatal("empty destination ID was accepted")
	}
	file := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewLocalPayloadPublisher("downloads", file, func(PayloadPublicationRequest) error { return nil }); err == nil {
		t.Fatal("non-directory publication root was accepted")
	}
}
