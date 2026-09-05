package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrBackupUnavailable = errors.New("backup service unavailable")

// BackupProtectionState is descriptive evidence supplied by the Backup
// authority. Sync may present or use this state for orchestration decisions, but
// it must not reinterpret the state as Sync authorization or delete Backup data.
type BackupProtectionState struct {
	Available        bool
	Protected        bool
	LatestCheckpoint string
	ObservedAt       time.Time
	Detail           string
}

// BackupProtectionStateProvider exposes Backup-owned protection state to Sync.
// The interface is intentionally read-only and contains no deletion operation.
type BackupProtectionStateProvider interface {
	ProtectionState(ctx context.Context, accountID, scopeID string) (BackupProtectionState, error)
}

type BackupCheckpointRequirement int

const (
	BackupCheckpointBestEffort BackupCheckpointRequirement = iota
	BackupCheckpointRequired
)

// BackupCheckpointRequest asks Backup to create a pre-change/pre-migration
// checkpoint for a Sync-owned scope. Backup remains authoritative for whether
// and how that checkpoint is created and retained.
type BackupCheckpointRequest struct {
	AccountID   string
	ScopeID     string
	OperationID string
	Reason      string
	Requirement BackupCheckpointRequirement
}

type BackupCheckpointReceipt struct {
	CheckpointID string
	CreatedAt    time.Time
}

// BackupCheckpointAuthorizer is a separate authorization boundary. A caller
// cannot create a checkpoint merely by being able to reach Backup.
type BackupCheckpointAuthorizer interface {
	AuthorizeCheckpoint(ctx context.Context, request BackupCheckpointRequest) error
}

// BackupCheckpointAuthority is deliberately limited to checkpoint creation.
// Backup deletion authority does not belong to Sync and is not represented here.
type BackupCheckpointAuthority interface {
	CreateCheckpoint(ctx context.Context, request BackupCheckpointRequest) (BackupCheckpointReceipt, error)
}

type BackupCheckpointOutcome struct {
	BackupAvailable bool
	Created         bool
	Receipt         BackupCheckpointReceipt
	Detail          string
}

// RestoreRequest identifies a Sync-managed restore target by logical authority
// identifiers rather than caller-supplied filesystem paths.
type RestoreRequest struct {
	AccountID   string
	TargetID    string
	OperationID string
}

// RestoreLease is issued by the Sync runtime after it has authorized the target,
// paused mutations for that target, and created an isolated staging area. The
// staging identifier is opaque; callers do not get destination-path authority.
type RestoreLease struct {
	LeaseID   string
	TargetID  string
	StagingID string
}

// SyncRestoreRuntime owns the Sync-side restore lifecycle for Sync-managed
// targets. It is responsible for target authorization, pause/maintenance state,
// isolated staging, reconciliation/publication, abort cleanup, and resume.
type SyncRestoreRuntime interface {
	BeginRestore(ctx context.Context, request RestoreRequest) (RestoreLease, error)
	CommitAndReconcile(ctx context.Context, lease RestoreLease) error
	AbortRestore(ctx context.Context, lease RestoreLease) error
	Resume(ctx context.Context, lease RestoreLease) error
}

// BackupSyncCoordinator composes Backup-owned protection/checkpoint authority
// with Sync-owned restore orchestration without transferring authority between
// the two services.
type BackupSyncCoordinator struct {
	Protection BackupProtectionStateProvider
	Checkpoint BackupCheckpointAuthority
	Authorizer BackupCheckpointAuthorizer
	Restore    SyncRestoreRuntime
}

// ProtectionState returns Backup's independent protection state. Backup
// unavailability is represented explicitly rather than being mistaken for an
// unprotected or protected state.
func (c BackupSyncCoordinator) ProtectionState(ctx context.Context, accountID, scopeID string) (BackupProtectionState, error) {
	if strings.TrimSpace(accountID) == "" {
		return BackupProtectionState{}, fmt.Errorf("account ID must not be empty")
	}
	if strings.TrimSpace(scopeID) == "" {
		return BackupProtectionState{}, fmt.Errorf("scope ID must not be empty")
	}
	if c.Protection == nil {
		return BackupProtectionState{Available: false, Detail: "backup protection state provider is not configured"}, nil
	}

	state, err := c.Protection.ProtectionState(ctx, accountID, scopeID)
	if errors.Is(err, ErrBackupUnavailable) {
		return BackupProtectionState{Available: false, Detail: err.Error()}, nil
	}
	if err != nil {
		return BackupProtectionState{}, err
	}
	if !state.Available {
		state.Protected = false
		state.LatestCheckpoint = ""
	}
	return state, nil
}

// CheckpointBeforeChange requests an authorized Backup checkpoint before a
// bounded change or migration. Required checkpoints fail closed when Backup is
// unavailable; best-effort checkpoints degrade explicitly and allow the caller
// to decide whether the underlying non-destructive operation may continue.
func (c BackupSyncCoordinator) CheckpointBeforeChange(ctx context.Context, request BackupCheckpointRequest) (BackupCheckpointOutcome, error) {
	if err := validateBackupCheckpointRequest(request); err != nil {
		return BackupCheckpointOutcome{}, err
	}
	if c.Authorizer == nil {
		return BackupCheckpointOutcome{}, fmt.Errorf("backup checkpoint authorizer is not configured")
	}
	if err := c.Authorizer.AuthorizeCheckpoint(ctx, request); err != nil {
		return BackupCheckpointOutcome{}, fmt.Errorf("authorize backup checkpoint: %w", err)
	}
	if c.Checkpoint == nil {
		if request.Requirement == BackupCheckpointRequired {
			return BackupCheckpointOutcome{}, fmt.Errorf("required backup checkpoint: %w", ErrBackupUnavailable)
		}
		return BackupCheckpointOutcome{
			BackupAvailable: false,
			Detail:          "backup checkpoint authority is not configured",
		}, nil
	}

	receipt, err := c.Checkpoint.CreateCheckpoint(ctx, request)
	if errors.Is(err, ErrBackupUnavailable) {
		if request.Requirement == BackupCheckpointRequired {
			return BackupCheckpointOutcome{}, fmt.Errorf("required backup checkpoint: %w", err)
		}
		return BackupCheckpointOutcome{BackupAvailable: false, Detail: err.Error()}, nil
	}
	if err != nil {
		return BackupCheckpointOutcome{}, err
	}
	if strings.TrimSpace(receipt.CheckpointID) == "" || receipt.CreatedAt.IsZero() {
		return BackupCheckpointOutcome{}, fmt.Errorf("backup returned an invalid checkpoint receipt")
	}
	return BackupCheckpointOutcome{
		BackupAvailable: true,
		Created:         true,
		Receipt:         receipt,
	}, nil
}

// RestoreIntoManagedTarget coordinates a restore into an already-authorized
// Sync-managed logical target. Restore bytes are written only through an opaque
// staging identifier issued by the Sync runtime. Publication happens only after
// successful staging and Sync-owned reconciliation. Resume is attempted on every
// begun lease, including failures.
func (c BackupSyncCoordinator) RestoreIntoManagedTarget(ctx context.Context, request RestoreRequest, restore func(context.Context, RestoreLease) error) (err error) {
	if err := validateRestoreRequest(request); err != nil {
		return err
	}
	if c.Restore == nil {
		return fmt.Errorf("sync restore runtime is not configured")
	}
	if restore == nil {
		return fmt.Errorf("restore callback must not be nil")
	}

	lease, err := c.Restore.BeginRestore(ctx, request)
	if err != nil {
		return err
	}
	if err := validateRestoreLease(request, lease); err != nil {
		abortErr := c.Restore.AbortRestore(ctx, lease)
		resumeErr := c.Restore.Resume(ctx, lease)
		return errors.Join(err, abortErr, resumeErr)
	}

	defer func() {
		err = errors.Join(err, c.Restore.Resume(ctx, lease))
	}()

	if err := restore(ctx, lease); err != nil {
		return errors.Join(err, c.Restore.AbortRestore(ctx, lease))
	}
	if err := c.Restore.CommitAndReconcile(ctx, lease); err != nil {
		return errors.Join(err, c.Restore.AbortRestore(ctx, lease))
	}
	return nil
}

func validateBackupCheckpointRequest(request BackupCheckpointRequest) error {
	if strings.TrimSpace(request.AccountID) == "" {
		return fmt.Errorf("account ID must not be empty")
	}
	if strings.TrimSpace(request.ScopeID) == "" {
		return fmt.Errorf("scope ID must not be empty")
	}
	if strings.TrimSpace(request.OperationID) == "" {
		return fmt.Errorf("operation ID must not be empty")
	}
	if strings.TrimSpace(request.Reason) == "" {
		return fmt.Errorf("checkpoint reason must not be empty")
	}
	if request.Requirement != BackupCheckpointBestEffort && request.Requirement != BackupCheckpointRequired {
		return fmt.Errorf("invalid backup checkpoint requirement")
	}
	return nil
}

func validateRestoreRequest(request RestoreRequest) error {
	if strings.TrimSpace(request.AccountID) == "" {
		return fmt.Errorf("account ID must not be empty")
	}
	if strings.TrimSpace(request.TargetID) == "" {
		return fmt.Errorf("restore target ID must not be empty")
	}
	if strings.TrimSpace(request.OperationID) == "" {
		return fmt.Errorf("operation ID must not be empty")
	}
	return nil
}

func validateRestoreLease(request RestoreRequest, lease RestoreLease) error {
	if strings.TrimSpace(lease.LeaseID) == "" {
		return fmt.Errorf("sync restore runtime returned an empty lease ID")
	}
	if lease.TargetID != request.TargetID {
		return fmt.Errorf("sync restore runtime returned a lease for the wrong target")
	}
	if strings.TrimSpace(lease.StagingID) == "" {
		return fmt.Errorf("sync restore runtime returned an empty staging ID")
	}
	return nil
}
