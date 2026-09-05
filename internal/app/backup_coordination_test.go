package app

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type fakeProtectionProvider struct {
	state BackupProtectionState
	err   error
}

func (f fakeProtectionProvider) ProtectionState(context.Context, string, string) (BackupProtectionState, error) {
	return f.state, f.err
}

type fakeCheckpointAuthorizer struct {
	err   error
	calls int
}

func (f *fakeCheckpointAuthorizer) AuthorizeCheckpoint(context.Context, BackupCheckpointRequest) error {
	f.calls++
	return f.err
}

type fakeCheckpointAuthority struct {
	receipt BackupCheckpointReceipt
	err     error
	calls   int
}

func (f *fakeCheckpointAuthority) CreateCheckpoint(context.Context, BackupCheckpointRequest) (BackupCheckpointReceipt, error) {
	f.calls++
	return f.receipt, f.err
}

type fakeRestoreRuntime struct {
	lease RestoreLease

	beginErr  error
	commitErr error
	abortErr  error
	resumeErr error

	calls []string
}

func (f *fakeRestoreRuntime) BeginRestore(context.Context, RestoreRequest) (RestoreLease, error) {
	f.calls = append(f.calls, "begin")
	return f.lease, f.beginErr
}

func (f *fakeRestoreRuntime) CommitAndReconcile(context.Context, RestoreLease) error {
	f.calls = append(f.calls, "commit")
	return f.commitErr
}

func (f *fakeRestoreRuntime) AbortRestore(context.Context, RestoreLease) error {
	f.calls = append(f.calls, "abort")
	return f.abortErr
}

func (f *fakeRestoreRuntime) Resume(context.Context, RestoreLease) error {
	f.calls = append(f.calls, "resume")
	return f.resumeErr
}

func TestBackupProtectionStatePreservesUnavailableAsUnknown(t *testing.T) {
	coordinator := BackupSyncCoordinator{
		Protection: fakeProtectionProvider{err: ErrBackupUnavailable},
	}

	state, err := coordinator.ProtectionState(context.Background(), "acct-1", "browser-state")
	if err != nil {
		t.Fatalf("ProtectionState returned error: %v", err)
	}
	if state.Available {
		t.Fatal("unavailable Backup must not be reported available")
	}
	if state.Protected {
		t.Fatal("unavailable Backup must not be reported protected")
	}
}

func TestBackupProtectionStateDoesNotInferProtectionWhenUnavailable(t *testing.T) {
	coordinator := BackupSyncCoordinator{
		Protection: fakeProtectionProvider{state: BackupProtectionState{
			Available:        false,
			Protected:        true,
			LatestCheckpoint: "checkpoint-should-not-survive",
		}},
	}

	state, err := coordinator.ProtectionState(context.Background(), "acct-1", "browser-state")
	if err != nil {
		t.Fatalf("ProtectionState returned error: %v", err)
	}
	if state.Protected || state.LatestCheckpoint != "" {
		t.Fatalf("unavailable state was not normalized: %+v", state)
	}
}

func TestCheckpointBeforeChangeRequiresIndependentAuthorization(t *testing.T) {
	denied := errors.New("checkpoint not authorized")
	authorizer := &fakeCheckpointAuthorizer{err: denied}
	backup := &fakeCheckpointAuthority{
		receipt: BackupCheckpointReceipt{CheckpointID: "cp-1", CreatedAt: time.Now()},
	}
	coordinator := BackupSyncCoordinator{Authorizer: authorizer, Checkpoint: backup}

	_, err := coordinator.CheckpointBeforeChange(context.Background(), BackupCheckpointRequest{
		AccountID:   "acct-1",
		ScopeID:     "browser-state",
		OperationID: "migration-1",
		Reason:      "schema migration",
		Requirement: BackupCheckpointRequired,
	})
	if !errors.Is(err, denied) {
		t.Fatalf("expected authorization error, got %v", err)
	}
	if authorizer.calls != 1 {
		t.Fatalf("expected one authorization call, got %d", authorizer.calls)
	}
	if backup.calls != 0 {
		t.Fatalf("Backup must not be called after denied authorization; got %d calls", backup.calls)
	}
}

func TestRequiredCheckpointFailsClosedWhenBackupUnavailable(t *testing.T) {
	authorizer := &fakeCheckpointAuthorizer{}
	backup := &fakeCheckpointAuthority{err: ErrBackupUnavailable}
	coordinator := BackupSyncCoordinator{Authorizer: authorizer, Checkpoint: backup}

	_, err := coordinator.CheckpointBeforeChange(context.Background(), BackupCheckpointRequest{
		AccountID:   "acct-1",
		ScopeID:     "sync-state",
		OperationID: "migration-1",
		Reason:      "pre-migration safety checkpoint",
		Requirement: BackupCheckpointRequired,
	})
	if !errors.Is(err, ErrBackupUnavailable) {
		t.Fatalf("expected Backup unavailability, got %v", err)
	}
}

func TestBestEffortCheckpointDegradesExplicitlyWhenBackupUnavailable(t *testing.T) {
	authorizer := &fakeCheckpointAuthorizer{}
	backup := &fakeCheckpointAuthority{err: ErrBackupUnavailable}
	coordinator := BackupSyncCoordinator{Authorizer: authorizer, Checkpoint: backup}

	outcome, err := coordinator.CheckpointBeforeChange(context.Background(), BackupCheckpointRequest{
		AccountID:   "acct-1",
		ScopeID:     "sync-state",
		OperationID: "change-1",
		Reason:      "non-destructive preference change",
		Requirement: BackupCheckpointBestEffort,
	})
	if err != nil {
		t.Fatalf("best-effort checkpoint returned error: %v", err)
	}
	if outcome.BackupAvailable || outcome.Created {
		t.Fatalf("unexpected optimistic outcome: %+v", outcome)
	}
}

func TestCheckpointBeforeChangeAcceptsValidBackupReceipt(t *testing.T) {
	createdAt := time.Unix(1_788_626_000, 0).UTC()
	authorizer := &fakeCheckpointAuthorizer{}
	backup := &fakeCheckpointAuthority{receipt: BackupCheckpointReceipt{
		CheckpointID: "cp-123",
		CreatedAt:    createdAt,
	}}
	coordinator := BackupSyncCoordinator{Authorizer: authorizer, Checkpoint: backup}

	outcome, err := coordinator.CheckpointBeforeChange(context.Background(), BackupCheckpointRequest{
		AccountID:   "acct-1",
		ScopeID:     "browser-state",
		OperationID: "migration-2",
		Reason:      "schema migration",
		Requirement: BackupCheckpointRequired,
	})
	if err != nil {
		t.Fatalf("CheckpointBeforeChange returned error: %v", err)
	}
	if !outcome.BackupAvailable || !outcome.Created || outcome.Receipt.CheckpointID != "cp-123" {
		t.Fatalf("unexpected checkpoint outcome: %+v", outcome)
	}
}

func TestRestoreIntoManagedTargetStagesCommitsReconcilesAndResumes(t *testing.T) {
	runtime := &fakeRestoreRuntime{lease: RestoreLease{
		LeaseID:   "lease-1",
		TargetID:  "browser-state",
		StagingID: "stage-1",
	}}
	coordinator := BackupSyncCoordinator{Restore: runtime}

	var received RestoreLease
	err := coordinator.RestoreIntoManagedTarget(context.Background(), RestoreRequest{
		AccountID:   "acct-1",
		TargetID:    "browser-state",
		OperationID: "restore-1",
	}, func(_ context.Context, lease RestoreLease) error {
		received = lease
		return nil
	})
	if err != nil {
		t.Fatalf("RestoreIntoManagedTarget returned error: %v", err)
	}
	if received.StagingID != "stage-1" {
		t.Fatalf("restore callback did not receive runtime staging lease: %+v", received)
	}
	wantCalls := []string{"begin", "commit", "resume"}
	if !reflect.DeepEqual(runtime.calls, wantCalls) {
		t.Fatalf("restore call order = %v, want %v", runtime.calls, wantCalls)
	}
}

func TestRestoreFailureAbortsBeforeResume(t *testing.T) {
	runtime := &fakeRestoreRuntime{lease: RestoreLease{
		LeaseID:   "lease-1",
		TargetID:  "browser-state",
		StagingID: "stage-1",
	}}
	coordinator := BackupSyncCoordinator{Restore: runtime}
	restoreErr := errors.New("backup restore failed")

	err := coordinator.RestoreIntoManagedTarget(context.Background(), RestoreRequest{
		AccountID:   "acct-1",
		TargetID:    "browser-state",
		OperationID: "restore-1",
	}, func(context.Context, RestoreLease) error {
		return restoreErr
	})
	if !errors.Is(err, restoreErr) {
		t.Fatalf("expected restore error, got %v", err)
	}
	wantCalls := []string{"begin", "abort", "resume"}
	if !reflect.DeepEqual(runtime.calls, wantCalls) {
		t.Fatalf("restore call order = %v, want %v", runtime.calls, wantCalls)
	}
}

func TestReconcileFailureAbortsBeforeResume(t *testing.T) {
	reconcileErr := errors.New("reconciliation failed")
	runtime := &fakeRestoreRuntime{
		lease:     RestoreLease{LeaseID: "lease-1", TargetID: "browser-state", StagingID: "stage-1"},
		commitErr: reconcileErr,
	}
	coordinator := BackupSyncCoordinator{Restore: runtime}

	err := coordinator.RestoreIntoManagedTarget(context.Background(), RestoreRequest{
		AccountID:   "acct-1",
		TargetID:    "browser-state",
		OperationID: "restore-1",
	}, func(context.Context, RestoreLease) error { return nil })
	if !errors.Is(err, reconcileErr) {
		t.Fatalf("expected reconciliation error, got %v", err)
	}
	wantCalls := []string{"begin", "commit", "abort", "resume"}
	if !reflect.DeepEqual(runtime.calls, wantCalls) {
		t.Fatalf("restore call order = %v, want %v", runtime.calls, wantCalls)
	}
}

func TestRestoreRejectsLeaseForDifferentTargetAndCleansUp(t *testing.T) {
	runtime := &fakeRestoreRuntime{lease: RestoreLease{
		LeaseID:   "lease-1",
		TargetID:  "different-target",
		StagingID: "stage-1",
	}}
	coordinator := BackupSyncCoordinator{Restore: runtime}
	called := false

	err := coordinator.RestoreIntoManagedTarget(context.Background(), RestoreRequest{
		AccountID:   "acct-1",
		TargetID:    "browser-state",
		OperationID: "restore-1",
	}, func(context.Context, RestoreLease) error {
		called = true
		return nil
	})
	if err == nil {
		t.Fatal("expected wrong-target lease to fail closed")
	}
	if called {
		t.Fatal("restore callback ran with unauthorized/mismatched target lease")
	}
	wantCalls := []string{"begin", "abort", "resume"}
	if !reflect.DeepEqual(runtime.calls, wantCalls) {
		t.Fatalf("restore call order = %v, want %v", runtime.calls, wantCalls)
	}
}
