# GoreeCloud Sync Operation Trust Checkpoints

## Scope

This document records the current source-level trust orchestration for reusing an authenticated secure peer. It is not a production-deployment or instantaneous-revocation claim.

## Authority boundary

GoreeCloud Identity and the account-scoped trusted-device store remain authoritative for device trust. The TLS 1.3 secure-peer layer remains responsible for authenticated transport. Higher-level Sync orchestration consumes those boundaries; it does not create trust independently.

## Single-operation guard

`RunWithCurrentTrust` revalidates the exact account, authenticated device ID, and bound key fingerprint immediately before one protected operation. If current trust cannot be established, the operation is not invoked and the local peer is closed by the fail-closed revalidation path.

## Bounded operation sequences

`RunOperationSequenceWithCurrentTrust` provides a reusable source-level orchestration seam for a finite sequence of protected peer operations.

The current contract:

- requires at least one operation;
- rejects sequences larger than 64 operations;
- validates every operation slot before executing the first callback;
- rejects a nil operation before any partial execution can occur;
- invokes the existing current-trust guard before every individual operation;
- stops at the first operation or trust-check error; and
- does not execute later operations after trust is revoked or otherwise ceases to be current between steps.

The bound exists to keep one synchronous orchestration request finite and reviewable. It is not a transport frame limit or a user-facing transfer-size limit.

## Revocation timing limitation

The sequence helper is synchronous and checkpoint-based. It does not monitor trust continuously, does not terminate every session at the instant a trust record changes, and cannot retroactively revoke an operation already running after its pre-operation checkpoint.

A future production runtime must still define approved lifecycle checkpoints, reconnect behavior, background revocation handling where required, Identity integration, and operational acceptance evidence.

## Acceptance state

This functionality is source-implemented and covered by repository tests and CI. It does not by itself establish production secure-listener orchestration, complete user-facing trust administration, Stable qualification, or production acceptance.
