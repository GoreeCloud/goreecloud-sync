# Features

This file distinguishes implemented repository capability from partial development capability and planned product capability. Source-level implementation does not imply production deployment or Stable qualification.

## Current source features

### Service and repository foundation

- Go module and first-party service/CLI shell.
- `serve`, `version`, and `help` commands.
- Loopback-only default development listener at `127.0.0.1:8787`.
- Development health and status endpoints.
- Privacy/security response headers and graceful process shutdown.
- GitHub Actions validation for formatting, tests, vetting, and build.

### Peer transport and identity foundations

- Pre-stabilization `GC-SYNC/1` capability handshakes.
- Bounded length-prefixed control framing with a 1 MiB hard limit and fail-closed malformed/truncated/unknown-field/trailing-value handling.
- Context-bound TCP peer transport helpers.
- Ed25519 device-key and fingerprint primitives.
- Signed pairing proofs bound to device identity and challenge.
- Cryptographically random, short-lived one-time pairing challenges.
- Exact challenge consumption with expiry and replay rejection.
- `VerifiedPairing` output only after pairing proof verification and challenge consumption.

### Trusted-device authorization foundations

- Durable account-scoped trusted-device records.
- Explicit authorization only from a verified pairing result.
- Explicit device revocation.
- Refusal to silently replace an active trusted device with different key material.
- Stored public-key/fingerprint consistency validation before trust is accepted.
- Private trusted-device storage directory/file permissions and atomic publication.
- Runtime trusted-peer resolver that composes authenticated peer resolution with exact account, device ID, and key-fingerprint authorization.
- Fail-closed handling for unknown, revoked, mismatched, or trust-store-error states at that resolver boundary.

### First-party application record replication foundations

- Dataset capability negotiation for GoreeCloud Browser, Search, and Bookmarks.
- Versioned transport-neutral record envelopes.
- Exact negotiated dataset/schema validation.
- Deterministic record conflict resolution.
- Payload-free tombstones for deletion propagation.
- Record-bound proof foundations and client/server proof conformance contracts.
- Replay/high-water, observation-receipt, tombstone-convergence, and protected device-key lifecycle foundations.
- Persistent development stores and source ingestion/retrieval handlers for `search.history`, `bookmarks.items`, `browser.tabs`, and `browser.history`.
- Authenticated retrieval ordered by record ID.
- 512-byte record-ID and cursor limit.
- Default retrieval page size of 256 and maximum of 1,024.
- Complete persisted-store validation before page return so off-page corruption remains fail-closed.
- Page-bounded candidate retention during retrieval rather than full-envelope page materialization.

## Partial or development-only features

- The default `serve` command exposes the base development service and does not by itself wire the complete replication, account, and trusted-device runtime composition.
- Pairing, one-time challenge consumption, trusted-device authorization, revocation, and trusted-peer enforcement exist as source foundations but do not yet provide complete user-facing pairing/trust administration.
- TCP peer transport exists, but authenticated encrypted peer-session establishment is not yet a completed product boundary.
- Account-scoped trusted-device state is durable locally, but production multi-user identity/account authority and runtime wiring remain incomplete.
- First-party application-record replication handlers exist, but they are not a complete folder synchronization engine and are not a production deployment claim.
- Privacy Shield and Wardveil decision boundaries exist in source, but complete production platform integration and acceptance evidence remain pending.

## Planned — Sync Core

- Persistent approved-folder replication.
- Send/receive, send-only, and receive-only modes.
- Selective synchronization.
- Filesystem watching and efficient change detection.
- Ignore patterns.
- Offline change queues.
- Chunked and resumable synchronization.
- Efficient changed-block transfer where justified.
- Bandwidth controls and schedules.
- File versioning.
- Conflict Center and user-facing conflict resolution.
- Deletion safety and recovery-aware controls.
- Whole-file and transfer integrity acceptance.
- Authoritative-copy labeling.
- Durable resume state and interrupted-transfer recovery.

## Planned — Nearby

- LAN discovery.
- User-facing QR pairing and optional confirmation workflows.
- File and folder transfer.
- Text, URL, and clipboard transfer.
- Drag and drop.
- Android share-sheet integration.
- Favorite devices and user-facing trust management.
- Background receiving.
- IPv4 and IPv6 operation.
- Wi-Fi, Ethernet, and approved private-network transport.
- Resume and post-transfer verification.

## Planned — Share

- Expiring links.
- One-use links.
- Download limits.
- Immediate share revocation.
- Optional passphrases.
- Optional authenticated-recipient requirements.
- QR codes.
- Client-side end-to-end encryption.
- Encrypted metadata where practical.
- Peer-to-peer preferred delivery.
- Encrypted temporary relay/storage fallback.
- Automatic expiration and cleanup.

## Planned — Product, multi-user, and administration

- Production GoreeCloud Identity/account runtime integration.
- Complete account and independently owned-device administration.
- User-facing device approval, revocation, recovery, retirement, and key-replacement workflows.
- Explicit folder ACLs and multi-user isolation tests.
- Read, write, contribute, receive-only, drop-only, and administrative grants.
- Audit events and administrative activity views.
- Transfer history.
- Migration Mode and drop folders.
- Glaze UI web administration and accessibility acceptance.
- Native Android client and Debian packaging.
- GoreeCloud Mesh coordination, GoreeCloud Network reachability, Notify, Manager, Photos, Drive, Search, API, Wardveil Security, Privacy Shield, Everkeep, and broader GoreeCloud Identity integrations as applicable.
- Reviewed authenticated encryption for peer sessions and transport/session replay protection beyond the implemented one-time pairing challenge.
- Production monitoring, deployment, migration/rollback, storage, and Stable acceptance evidence.
