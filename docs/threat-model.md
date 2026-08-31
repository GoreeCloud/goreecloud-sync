# GoreeCloud Sync Threat Model

**Status:** Active Development — pre-Stable.

## Purpose

This document defines the current GoreeCloud Sync threat model. It distinguishes source controls that now exist from controls that still require production implementation and acceptance evidence.

GoreeCloud Sync combines persistent synchronization, nearby transfer, and temporary encrypted sharing. Network reachability, transport authentication, durable device trust, account authorization, dataset/folder authorization, transfer integrity, and recovery remain separate security boundaries.

## Security invariants

1. Network reachability never grants GoreeCloud Sync authorization.
2. A successful TLS connection never grants dataset, folder, transfer, share, or administrative authorization by itself.
3. Every trusted device has an individual key identity and revocable lifecycle.
4. Pairing proof is not durable authorization; explicit trusted-device approval is required.
5. Capability handshakes must not override the device identity authenticated by a secure peer connection.
6. Folder and transfer access must be explicit and least privilege.
7. A completed byte transfer is not considered verified until required integrity validation succeeds.
8. Synchronization is not backup or recovery authority.
9. Secrets, reusable credentials, private keys, and private transfer contents must not enter source control or ordinary diagnostic logs.
10. Temporary sharing must not weaken persistent-sync authorization boundaries.
11. Cryptographic transport must use established, reviewed libraries and protocols rather than custom encryption.

## Protected assets

- User accounts and account-to-device ownership relationships.
- Device identities, protected private keys, pairing state, trusted-device records, and revocation state.
- Authenticated peer-session identity and connection-admission decisions.
- Dataset and folder authorization and synchronization-direction policy.
- Application records, file contents, filenames, metadata, hashes, and transfer manifests.
- Temporary share secrets and encrypted temporary payloads.
- Conflict versions and deletion/version history.
- Audit and security events.
- Administrative configuration and service state.

## Trust boundaries

### Client device boundary

A device may be lost, stolen, compromised, malicious, or incorrectly assigned. Device approval must therefore be explicit, key-bound, independently revocable, and recoverable without silently trusting replacement key material.

Current source includes one-time pairing challenges, signed Ed25519 pairing proofs, durable account-scoped trusted-device records, explicit revocation, and active-key replacement protection. Complete user-facing approval/recovery/key-replacement workflows remain pending.

### Network and peer-transport boundary

LAN, Internet, GoreeCloud Network, NetBird connectivity, and raw TCP reachability provide transport only. They do not imply application trust.

Current source includes a separate TLS 1.3 peer primitive for already-trusted peers. Both sides prove possession of their Ed25519 device key, and the remote connection is pinned to an expected canonical device ID and exact raw public key supplied by higher-level trust state. The GC-SYNC capability handshake is then required to claim the same authenticated device identity.

Raw `DialPeer` and `AcceptPeer` remain unauthenticated lower-level primitives and must not be used as a secure-session substitute.

### Service and authorization boundary

The GoreeCloud Sync service must authenticate callers and authorize each protected operation independently of network location or transport success.

Current source includes authenticated peer/session models, trusted-device enforcement at a peer-resolver boundary, dataset capability negotiation, and Privacy Shield/Wardveil-facing record-acceptance boundaries. Production multi-user account/folder policy and complete runtime composition remain pending.

### Storage boundary

The service must receive access only to approved datasets and later approved folders. Container-local or broadly mounted host storage must not become an implicit data authority.

Current application-record stores validate persisted state before returning synchronized records. Production folder synchronization still requires root confinement, symlink policy, safe temporary writes, and storage-isolation acceptance.

### Temporary relay/share boundary

A relay or temporary-share service must be treated as potentially observable infrastructure. End-to-end encryption is required where GoreeCloud Share claims relay-independent confidentiality.

The direct TLS peer primitive does not satisfy future Share E2EE requirements for relay-backed delivery.

## Primary threat classes

### Unauthorized device enrollment

Threats include stolen pairing material, unattended approval, social engineering, replay, and device impersonation.

Implemented source controls include cryptographically random short-lived pairing challenges, exact single consumption, expiry/replay rejection, signed key-possession proofs, and explicit conversion of verified pairing state into durable trusted-device authorization.

Still required: user-facing human-verifiable approval, enrollment rate limits, security events, recovery UX, and production Identity integration.

### Device or key impersonation

An attacker may present a different key while claiming a trusted device ID, substitute a certificate, or claim a different device in the protocol handshake.

Implemented source controls include exact trusted raw-key pinning in the TLS 1.3 secure-peer wrapper, canonical device-ID binding, mutual TLS CertificateVerify possession proof, GoreeCloud Sync ALPN, and post-TLS GC-SYNC handshake identity matching.

Still required: production trusted-device lookup/orchestration, revocation-aware reconnect policy, operational telemetry, and deployment acceptance.

### Downgrade or insecure transport substitution

An attacker or incorrect caller may attempt to use older TLS, a non-GoreeCloud application protocol, or raw TCP where an authenticated encrypted connection is required.

The secure-peer wrapper requires TLS 1.3 only and GoreeCloud Sync ALPN. Raw transport remains deliberately separate so higher-level code can deny it for security-sensitive paths rather than treating it as implicitly upgraded.

Still required: production listener/dial policy that selects secure transport for every protected peer path, plus deployment tests proving insecure fallback is not accepted.

### Compromised or lost device

A previously trusted device may become hostile.

Implemented source controls include independent trusted-device revocation and runtime trust checks that reject revoked or mismatched device fingerprints when that resolver is wired.

Still required: bounded cached authorization, reconnect invalidation, last-seen/trust state, recovery and replacement workflows, and audit/security events.

### Authorization bypass

An authenticated peer may attempt to read, write, delete, contribute to, or administer data beyond its permissions.

Implemented source foundations include dataset capability negotiation, exact schema/dataset validation, authenticated peer resolution, and server-side Privacy Shield/Wardveil acceptance boundaries.

Still required: production account/folder roles, deny-by-default policy composition, drop-only isolation, complete multi-user isolation tests, and administrative authorization.

### Replay and stale operations

An attacker or delayed peer may replay pairing, record, deletion, transfer, or authorization operations.

Implemented source controls include one-time pairing-challenge replay rejection and record replay/high-water foundations. TLS session tickets are disabled in the current secure-peer primitive.

Still required: complete transport/session freshness semantics, operation-specific replay policy, stale authorization invalidation, resumable-transfer sequence semantics, and end-to-end acceptance tests.

### Transfer tampering and corruption

Data may be corrupted accidentally or modified in transit.

The secure-peer primitive provides TLS 1.3 transport confidentiality and integrity for already-trusted direct peers. Application records also use validated envelopes and proof foundations.

Still required: chunk/file integrity for folder transfers, atomic completion, interrupted-transfer recovery, and an explicit distinction between transmitted and verified file state.

### Path traversal and filesystem escape

Hostile names, symlinks, mount boundaries, archive contents, or normalization differences may attempt to escape an approved synchronization root.

Folder synchronization is not yet complete; production implementation must include canonical path validation, root confinement, symlink policy, filename portability checks, safe temporary-file handling, and adversarial tests before Stable.

### Conflict-driven data loss

Concurrent writes may cause silent overwrite or destructive convergence.

Current application-record foundations provide deterministic record conflict behavior and tombstone convergence. Future folder synchronization still requires conflict preservation, user-visible Conflict Center workflows, and no silent destructive resolution.

### Deletion propagation

Accidental or malicious deletion may replicate to other endpoints.

Current application-record deletion uses payload-free tombstones with convergence controls. Future folder deletion requires documented propagation semantics, visibility, versioning/recovery behavior where appropriate, and independent recovery protection for important data.

### Resource exhaustion and abuse

Peers may consume CPU, memory, storage, file descriptors, bandwidth, handshake capacity, or connection slots.

Current source includes bounded control frames, bounded retrieval, bounded TLS handshake duration, and cancellation-aware transport foundations.

Still required: connection concurrency controls, quotas, rate limits, backpressure, bounded retries, transfer/storage limits, cleanup, and adversarial resource-exhaustion acceptance.

### Metadata and privacy leakage

Logs, discovery, share metadata, filenames, device names, certificates, or diagnostics may reveal sensitive information.

Required controls include data minimization, privacy-conscious diagnostics, no payload/private-key/reusable-secret logging, bounded retention, and encrypted metadata for temporary shares where justified.

### Temporary share compromise

Share links may be guessed, leaked, reused, retained beyond expiration, or downloaded more times than authorized.

Required future controls include high-entropy secrets, expiration, one-use/download limits, immediate revocation, optional passphrases or authenticated recipients, automatic cleanup, and end-to-end encryption where claimed.

### Supply-chain compromise

Dependencies, build tooling, or release artifacts may be compromised.

Required controls include dependency review, vulnerability scanning, traceable builds, exact-head CI, release provenance, secret separation, and exact-release acceptance.

## Explicitly prohibited assumptions

- A trusted LAN is not sufficient authentication.
- GoreeCloud Network or NetBird membership is not sufficient GoreeCloud Sync authorization.
- A raw TCP connection is not an authenticated Sync session.
- A self-signed certificate from an unknown key is not trusted merely because TLS succeeds.
- A successful pinned TLS session is not dataset, folder, transfer, or administrative authorization.
- A GC-SYNC handshake may not claim a device identity different from the secure transport identity.
- Possession of a share URL must not silently grant broader account or folder access.
- Successful byte transmission is not proof of content integrity.
- A synchronized replica is not automatically a backup.
- A server administrator should not be assumed unable to observe relay plaintext unless Share E2EE actually provides that property.

## Required security validation before Stable

At minimum, production-readiness evidence must cover:

- multi-user isolation and server-side authorization;
- device enrollment, revocation, recovery, and key replacement;
- secure peer connection admission, key substitution, identity mismatch, downgrade, and revocation behavior;
- replay/stale-operation handling across pairing, sessions, records, deletions, and transfers;
- folder authorization and drop-only isolation;
- path traversal and symlink escape resistance;
- transfer integrity and interrupted-transfer recovery;
- conflict and deletion safety;
- Share expiration, revocation, download limits, and E2EE behavior where claimed;
- rate limiting, bounded concurrency, and resource exhaustion;
- logging privacy and secret separation;
- dependency vulnerability review;
- migration and rollback behavior;
- packaging, deployment, and exact-release validation;
- applicable Wardveil Security, Privacy Shield, Everkeep, GoreeCloud Identity, GoreeCloud Mesh, and Glaze UI acceptance evidence.

## Current claim boundary

GoreeCloud Sync now contains tested source controls for pairing replay/expiry, durable trusted-device authorization, trust-enforced peer resolution, bounded application-record replication, and a TLS 1.3 authenticated direct-peer primitive for already-trusted device identities. Complete production session orchestration, folder synchronization, multi-user authorization, Nearby, Share, deployment, and Stable security acceptance remain incomplete.
