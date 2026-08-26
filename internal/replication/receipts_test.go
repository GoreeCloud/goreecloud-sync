package replication

import (
	"testing"
	"time"

	"github.com/GoreeCloud/goreecloud-sync/internal/datasets"
	"github.com/GoreeCloud/goreecloud-sync/internal/session"
)

func TestConvergenceRequiresEveryPeerReceipt(t *testing.T) {
	record := datasets.RecordEnvelope{Dataset: "search.history", SchemaVersion: 1, RecordID: "gone", Revision: 3, UpdatedAt: time.Unix(500, 0).UTC(), OriginDevice: "device-a", Deleted: true}
	peerA := session.AuthenticatedPeer{DeviceID: "device-a"}
	peerB := session.AuthenticatedPeer{DeviceID: "device-b"}
	receiptA, err := NewObservationReceipt(record, peerA, time.Unix(501, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if ConvergenceConfirmed(record, []string{"device-a", "device-b"}, []ObservationReceipt{receiptA}) {
		t.Fatal("convergence confirmed before every required peer observed tombstone")
	}
	receiptB, err := NewObservationReceipt(record, peerB, time.Unix(502, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if !ConvergenceConfirmed(record, []string{"device-a", "device-b"}, []ObservationReceipt{receiptA, receiptB}) {
		t.Fatal("expected convergence after all peer receipts")
	}
}

func TestConvergenceRejectsReceiptForDifferentRevision(t *testing.T) {
	record := datasets.RecordEnvelope{Dataset: "search.history", SchemaVersion: 1, RecordID: "gone", Revision: 4, UpdatedAt: time.Unix(600, 0).UTC(), OriginDevice: "device-a", Deleted: true}
	receipt, err := NewObservationReceipt(record, session.AuthenticatedPeer{DeviceID: "device-a"}, time.Unix(601, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	receipt.Revision = 3
	if ConvergenceConfirmed(record, []string{"device-a"}, []ObservationReceipt{receipt}) {
		t.Fatal("mismatched revision receipt confirmed convergence")
	}
}
