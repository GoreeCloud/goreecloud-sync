package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrBackupUnavailable = errors.New("backup service unavailable")

const restoreCleanupTimeout = 5 * time.Second

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

// BackupCheckpointReceipt binds Backup's result to the exact authorized request
// so a receipt for another account, scope, or operation cannot be replayed as
// evidence for this checkpoint request.
type BackupCheckpointReceipt struct {
	CheckpointID string
	AccountID    string
	ScopeID      string
	OperationID  string
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
// unprotected or protected state. Available evidence must carry a real,
// non-future observation time so Sync cannot present undated or future-dated
// Backup state as current evidence.
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
		state.ObservedAt = time.Time{}
		return state, nil
	}
	if state.ObservedAt.IsZero() {
		return BackupProtectionState{}, fmt.Errorf("available backup protection state is missing observed time")
	}
	if state.ObservedAt.After(time.Now().UTC()) {
		return BackupProtectionState{}, fmt.Errorf("backup protection state cannot be observed in the future")
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
	if err := validateBackupCheckpointReceipt(request, receipt); err != nil {
		return BackupCheckpointOutcome{}, err
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
// successful staging and Sync-owned reconciliation. Once a lease has begun,
// abort and resume cleanup each receive their own bounded cleanup context that
// survives cancellation of the caller's request context.
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
		abortErr := runRestoreCleanup(ctx, func(cleanupCtx context.Context) error {
			return c.Restore.AbortRestore(cleanupCtx, lease)
		})
		resumeErr := runRestoreCleanup(ctx, func(cleanupCtx context.Context) error {
			return c.Restore.Resume(cleanupCtx, lease)
		})
		return errors.Join(err, abortErr, resumeErr)
	}

	defer func() {
		resumeErr := runRestoreCleanup(ctx, func(cleanupCtx context.Context) error {
			return c.Restore.Resume(cleanupCtx, lease)
		})
		err = errors.Join(err, resumeErr)
	}()

	if err := restore(ctx, lease); err != nil {
		abortErr := runRestoreCleanup(ctx, func(cleanupCtx context.Context) error {
			return c.Restore.AbortRestore(cleanupCtx, lease)
		})
		return errors.Join(err, abortErr)
	}
	if err := c.Restore.CommitAndReconcile(ctx, lease); err != nil {
		abortErr := runRestoreCleanup(ctx, func(cleanupCtx context.Context) error {
			return c.Restore.AbortRestore(cleanupCtx, lease)
		})
		return errors.Join(err, abortErr)
	}
	return nil
}

func runRestoreCleanup(parent context.Context, operation func(context.Context) error) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(parent), restoreCleanupTimeout)
	defer cancel()
	return operation(cleanupCtx)
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

func validateBackupCheckpointReceipt(request BackupCheckpointRequest, receipt BackupCheckpointReceipt) error {
	if strings.TrimSpace(receipt.CheckpointID) == "" || receipt.CreatedAt.IsZero() {
		return fmt.Errorf("backup returned an invalid checkpoint receipt")
	}
	if receipt.CreatedAt.After(time.Now().UTC()) {
		return fmt.Errorf("backup checkpoint receipt cannot be created in the future")
	}
	if receipt.AccountID != request.AccountID {
		return fmt.Errorf("backup checkpoint receipt account does not match the authorized request")
	}
	if receipt.ScopeID != request.ScopeID {
		return fmt.Errorf("backup checkpoint receipt scope does not match the authorized request")
	}
	if receipt.OperationID != request.OperationID {
		return fmt.Errorf("backup checkpoint receipt operation does not match the authorized request")
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
