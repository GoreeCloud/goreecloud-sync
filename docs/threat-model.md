# GoreeCloud Sync Threat Model

Status: Milestone 0 development record

## Purpose

This document defines the initial security threat model for GoreeCloud Sync. It is a development baseline, not a claim of production security acceptance.

GoreeCloud Sync combines persistent synchronization, nearby transfer, and temporary encrypted sharing. The security model therefore treats network reachability, device trust, user authorization, folder authorization, transfer integrity, and recovery as separate concerns.

## Security invariants

1. Network reachability never grants GoreeCloud Sync authorization.
2. Every enrolled device has an individual identity and lifecycle.
3. Folder and transfer access is explicit and least-privilege.
4. A completed transfer is not considered verified until required integrity validation succeeds.
5. Synchronization is not backup authority.
6. Secrets, reusable credentials, private keys, and private transfer contents must not enter source control or ordinary diagnostic logs.
7. Temporary sharing must not weaken persistent-sync authorization boundaries.
8. Cryptographic implementation must use established, reviewed libraries and protocols rather than custom primitives.

## Protected assets

- User accounts and account-to-device ownership relationships.
- Device identities, private keys, pairing state, and revocation state.
- Folder authorization and synchronization-direction policies.
- File contents, filenames, metadata, hashes, and transfer manifests.
- Temporary share secrets and encrypted temporary payloads.
- Conflict versions and deletion/version history.
- Audit and security events.
- Administrative configuration and service state.

## Trust boundaries

### Client device boundary

A client may be lost, stolen, compromised, malicious, or incorrectly assigned. Device enrollment must therefore be explicit and revocable.

### Network boundary

LAN, Internet, GoreeCloud Network, and NetBird connectivity provide transport only. They do not imply application trust.

### Service boundary

The GoreeCloud Sync service must authenticate callers and authorize each protected operation independently of network location.

### Storage boundary

The service must receive access only to approved directories or datasets. Container-local or broadly mounted host storage must not become an implicit data authority.

### Temporary relay boundary

A relay or temporary-share service must be treated as potentially observable infrastructure. End-to-end encryption is required where the product claims relay-independent confidentiality.

## Primary threat classes

### Unauthorized device enrollment

Threats include stolen pairing material, unattended approval, social engineering, replay, and device impersonation.

Required controls include explicit approval, short-lived pairing material, human-verifiable device identity, rate limiting, replay resistance, revocation, and security events.

### Compromised or lost device

A previously trusted device may become hostile.

Required controls include independent device revocation, key replacement, bounded cached authorization, clear last-seen/trust state, and removal without requiring deletion of the user's account.

### Authorization bypass

An authenticated user or device may attempt to read, write, delete, contribute to, or administer data beyond its permissions.

Required controls include server-side authorization, deny-by-default policy evaluation, explicit folder roles, drop-only separation, and multi-user isolation tests.

### Path traversal and filesystem escape

Hostile names, symlinks, mount boundaries, archive contents, or normalization differences may attempt to escape an approved synchronization root.

Required controls include canonical path validation, root confinement, symlink policy, filename portability checks, safe temporary-file handling, and adversarial tests.

### Transfer tampering and corruption

Data may be corrupted accidentally or modified in transit.

Required controls include authenticated encrypted transport, per-chunk verification where used, whole-file verification where appropriate, atomic completion, and explicit distinction between transmitted and verified states.

### Replay and stale operations

An attacker or delayed peer may replay old transfer, deletion, pairing, or authorization operations.

Required controls include unique operation identifiers, freshness or sequence semantics where appropriate, idempotency rules, and rejection of invalid stale state transitions.

### Conflict-driven data loss

Concurrent writes may cause silent overwrite or destructive convergence.

Required controls include conflict preservation, deterministic conflict detection, user-visible Conflict Center workflows, and no silent destructive resolution.

### Deletion propagation

Accidental or malicious deletion may replicate to other endpoints.

Required controls include documented deletion semantics, versioning/recovery controls where appropriate, propagation visibility, and independent backup protection for important folders.

### Resource exhaustion and abuse

Peers may consume CPU, memory, storage, file descriptors, bandwidth, or connection slots.

Required controls include size and concurrency limits, quotas where appropriate, bounded retries, rate limits, cancellation, backpressure, temporary-storage limits, and cleanup.

### Metadata and privacy leakage

Logs, discovery, share metadata, filenames, device names, or diagnostics may reveal sensitive information.

Required controls include data minimization, privacy-conscious diagnostics, no payload logging, bounded retention, and encrypted filenames/metadata for temporary shares where practical.

### Temporary share compromise

Share links may be guessed, leaked, reused, retained beyond expiration, or downloaded more times than authorized.

Required controls include high-entropy secrets, expiration, one-use/download limits, immediate revocation, optional passphrases/authenticated recipients, automatic cleanup, and end-to-end encryption where claimed.

### Supply-chain compromise

Dependencies, build tooling, or release artifacts may be compromised.

Required controls include dependency review, vulnerability scanning, reproducible or traceable build practices where practical, exact-head CI, release provenance, and secret separation.

## Explicitly prohibited assumptions

- A trusted LAN is not sufficient authentication.
- NetBird membership is not sufficient GoreeCloud Sync authorization.
- Possession of a share URL must not silently grant broader account or folder access.
- Successful byte transmission is not proof of integrity.
- A synchronized replica is not automatically a backup.
- A server administrator should not be assumed invisible to plaintext relay data unless end-to-end encryption actually provides that property.

## Required security validation before Stable

At minimum, production-readiness evidence must cover:

- Multi-user isolation.
- Device enrollment and revocation.
- Folder authorization and drop-only isolation.
- Path traversal and symlink escape resistance.
- Transfer integrity and interrupted-transfer recovery.
- Replay/stale-operation behavior.
- Conflict and deletion safety.
- Share expiration, revocation, download limits, and E2EE behavior where claimed.
- Rate limiting and resource exhaustion.
- Logging privacy and secret separation.
- Dependency vulnerability review.
- Migration and rollback behavior.

## Milestone 0 boundary

This threat model defines requirements for implementation and testing. It does not claim that the controls above are already implemented. Milestone 0 remains a foundation stage until its required source, design, CI, Glaze UI shell, and security-development records are reviewed and integrated.