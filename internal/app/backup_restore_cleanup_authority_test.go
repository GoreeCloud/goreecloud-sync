package app

import (
	"context"
	"testing"
)

type cleanupLeaseRecordingRuntime struct {
	lease       RestoreLease
	abortLease  RestoreLease
	resumeLease RestoreLease
}

func (r *cleanupLeaseRecordingRuntime) BeginRestore(context.Context, RestoreRequest) (RestoreLease, error) {
	return r.lease, nil
}

func (r *cleanupLeaseRecordingRuntime) CommitAndReconcile(context.Context, RestoreLease) error {
	return nil
}

func (r *cleanupLeaseRecordingRuntime) AbortRestore(_ context.Context, lease RestoreLease) error {
	r.abortLease = lease
	return nil
}

func (r *cleanupLeaseRecordingRuntime) Resume(_ context.Context, lease RestoreLease) error {
	r.resumeLease = lease
	return nil
}

func TestRestoreCleanupDoesNotReuseForeignLeaseAuthority(t *testing.T) {
	request := RestoreRequest{
		AccountID:   "acct-1",
		TargetID:    "browser-state",
		OperationID: "restore-1",
	}
	runtime := &cleanupLeaseRecordingRuntime{lease: RestoreLease{
		LeaseID:     "foreign-lease",
		AccountID:   "acct-2",
		TargetID:    "different-target",
		OperationID: "different-operation",
		StagingID:   "foreign-stage",
	}}
	coordinator := BackupSyncCoordinator{Restore: runtime}
	called := false

	err := coordinator.RestoreIntoManagedTarget(context.Background(), request, func(context.Context, RestoreLease) error {
		called = true
		return nil
	})
	if err == nil {
		t.Fatal("foreign restore lease must fail closed")
	}
	if called {
		t.Fatal("restore callback ran with a foreign restore lease")
	}

	for name, cleanupLease := range map[string]RestoreLease{
		"abort":  runtime.abortLease,
		"resume": runtime.resumeLease,
	} {
		if cleanupLease.AccountID != request.AccountID || cleanupLease.TargetID != request.TargetID || cleanupLease.OperationID != request.OperationID {
			t.Fatalf("%s cleanup escaped request authority: %+v", name, cleanupLease)
		}
		if cleanupLease.LeaseID != "" || cleanupLease.StagingID != "" {
			t.Fatalf("%s cleanup reused foreign lease/staging authority: %+v", name, cleanupLease)
		}
	}
}
