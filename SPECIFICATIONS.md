# GoreeCloud Sync Specifications

## Status

**Lifecycle:** Active Development — pre-Stable.

GoreeCloud Sync is original GoreeCloud-owned software for private synchronization and secure transfer. Current source implements development foundations for first-party application-record replication, raw and TLS-authenticated peer transport, device identity, pairing proof/challenge handling, durable trusted-device authorization, fail-closed current-trust checks, transfer manifests and integrity verification, and a bounded authenticated one-to-one file/text payload stream with explicit receiver authorization. It is not a complete production synchronization product and is not approved as Stable.

## Purpose and scope

GoreeCloud Sync is intended to provide three related product modes while keeping their authorization and security boundaries explicit:

- **Sync** — persistent replication of explicitly approved application datasets and, later, approved folders.
- **Nearby** — direct device-to-device transfer over local or approved private-network paths.
- **Share** — temporary encrypted delivery to recipients, including expiring links where implemented later.

Synchronization does not establish backup authority. Everkeep remains the GoreeCloud continuity, backup, preservation, and recovery authority.

## Implemented source foundations

Current source includes:

- a Go service and CLI shell with loopback-safe development defaults, health/status endpoints, and graceful shutdown;
- pre-stabilization `GC-SYNC/1` capability handshakes, bounded control framing, bounded binary chunk framing, and context-bound raw TCP peer helpers;
- a separate TLS 1.3-only mutually authenticated secure-peer wrapper for already-trusted devices, using Go's reviewed `crypto/tls` and `crypto/x509` implementations;
- short-lived self-signed Ed25519 certificates used as TLS key-possession carriers, with exact expected device-ID/raw-public-key pinning, `goreecloud-sync/1` ALPN binding, bounded TLS handshakes, session-ticket disabling, mutual acceptance confirmation, and GC-SYNC capability-handshake identity binding;
- Ed25519 device identity, key fingerprinting, and signed pairing proofs;
- cryptographically random, short-lived one-time pairing challenges with expiry and replay rejection;
- `VerifiedPairing` output only after proof verification and exact challenge consumption;
- a hardened local device-key store with canonical IDs, consistent Ed25519 material, protected private-key opening, owner-only persistence, and atomic rotation;
- a durable account-scoped trusted-device registry with explicit revocation, atomic/private persistence, stored-key/fingerprint validation, and active-key replacement protection;
- a trusted peer resolver requiring the exact account/device/key fingerprint to remain authorized;
- an application-layer secure-peer factory that resolves current non-revoked remote trust before opening the local key, establishes the existing TLS 1.3 session, binds the fingerprint of the admitted trusted public key to the resulting peer, and provides explicit current-trust revalidation checkpoints;
- `RunWithCurrentTrust` and bounded operation-sequence orchestration that can recheck exact account/device/key trust before protected operations;
- versioned single-file manifests with ordered per-chunk and whole-payload SHA-256 metadata, current 1 MiB chunk bounds, and fail-closed corruption/truncation/trailing-data verification;
- versioned one-to-one file/text payload offers, receiver decisions, sender completion records, verified receiver receipts, and random 128-bit transfer IDs;
- an authenticated payload stream that refuses raw or trust-unbound peers, requires explicit receiver application authorization before bytes are sent, verifies source chunks before transmission, verifies received chunks before staging, and closes the peer on protocol/stream/integrity failures;
- application-level payload-transfer composition that revalidates current durable trust before transfer start, before source reads, before staging writes, and before returning verified success;
- first-party dataset capability negotiation and replication handlers for `search.history`, `bookmarks.items`, `browser.tabs`, and `browser.history`;
- transport-neutral record envelopes, deterministic conflict resolution, payload-free tombstones, record-bound proof foundations, replay/high-water foundations, observation receipts, and tombstone-convergence controls;
- authenticated bounded retrieval ordered by record ID, with a 512-byte record/cursor bound, default page size 256, maximum page size 1,024, full persisted-store validation, and page-bounded candidate retention.

These are source and development contracts. The default development `serve` command does not by itself enable secure peer listeners/dials, payload-transfer runtime orchestration, discovery/address policy, production account authority, production revalidation scheduling, revocation-aware reconnect policy, durable resume state, or full secure-session lifecycle management.

## Architecture

Primary repository areas are:

- `cmd/goreecloud-sync/` — CLI and service entry point.
- `internal/app/` — HTTP composition, ingestion/retrieval, authenticated peer resolution, trust enforcement, secure-peer identity composition, explicit trust revalidation, and payload-transfer trust composition.
- `internal/identity/` — device keys, pairing proofs, one-time challenges, trusted-device authorization, fingerprints, and validation.
- `internal/session/` — authenticated peer/session models and session-bound identity state.
- `internal/transport/` — pre-stabilization framing/handshake, raw TCP helpers, TLS 1.3 authenticated secure-peer helpers, bounded binary chunk frames, and the one-to-one payload stream.
- `internal/transfer/` — chunk/session models, manifests, integrity verification, transfer identifiers, and payload control records.
- `internal/datasets/` — capabilities, record envelopes, validation, and conflict behavior.
- `internal/replication/` — persistence, ingestion state, retrieval, replay/receipt/tombstone foundations.
- `internal/policy/` — Privacy Shield and Wardveil-facing acceptance boundaries.
- `protocol/` — pre-stabilization protocol records and interoperability fixtures.
- `docs/` — architecture, security, threat-model, and engineering documentation.

Application semantics remain owned by the originating GoreeCloud application. Sync coordinates authorized replication and must not become an implicit authority over Browser, Search, Bookmarks, or other application data.

## Peer transport and payload boundary

Raw `DialPeer` and `AcceptPeer` establish/wrap context-bound TCP streams and intentionally assign no trust.

`DialSecurePeer` and `AcceptSecurePeer` add TLS 1.3 authentication/encryption for a peer whose exact identity is already trusted. The transport requires exact device identity and Ed25519 key pinning, mutual certificate possession, GoreeCloud Sync ALPN, bounded handshake time, accepting-side confirmation, and later GC-SYNC identity consistency.

`SecurePeerFactory` is the application trust composition layer. For each explicit dial/accept call it resolves the requested remote device from current account-scoped durable trust before opening the local private key, rejects revoked/unknown/cross-account/corrupt trust, and invokes the TLS primitive with those exact identities. After secure admission it binds the fingerprint of that exact validated remote public key to `PeerConn`.

`SecurePeerFactory.RevalidatePeer` is an explicit current-authorization checkpoint. It compares the authenticated peer device ID and bound key fingerprint with the current account-scoped `TrustedDeviceStore`. A revoked, replaced, missing, corrupt, or unreadable trust record fails closed and closes the local peer.

The Development payload transport builds on that boundary. A `PeerConn` cannot perform the payload operation unless it has both a TLS-authenticated device identity and the durable trusted-device fingerprint bound by the application trust layer. The receiver must separately authorize the offered file/text operation before an acceptance decision is sent and before any payload bytes move.

Payload control records are bounded JSON frames. Payload chunks are separate binary length-prefixed frames whose declared size is checked against both the 1 MiB global frame ceiling and the exact expected manifest chunk size before allocation. The sender emits a completion record only after independently rechecking the source against the manifest; the receiver emits a verified receipt only after all chunks and the complete payload hash are verified.

The receiver writes only to a caller-provided staging destination. A verified return authorizes the caller to consider the staged bytes integrity-checked for that transfer; it does not independently authorize a filesystem destination, overwrite, rename, conflict resolution, retention, backup, or recovery action.

Current-trust checks remain checkpoint-based. Payload application composition revalidates before transfer start, source reads, staging writes, and verified return, but no claim is made that every established connection or already-running I/O is stopped at the exact instant trust changes.

## Authentication, authorization, and trust

A network connection is not application authorization.

Current source separates these trust steps:

1. A device proves possession of its Ed25519 key with a signed pairing proof.
2. The proof consumes the exact short-lived challenge; expired/reused challenges fail.
3. Explicit application logic may authorize the verified pairing into an account-scoped durable trusted-device store.
4. Authenticated application access may require current account/device/fingerprint authorization.
5. Direct secure transport resolves current durable trust and protected local identity before TLS admission.
6. TLS proves exact key possession/identity and the GC-SYNC handshake must claim those same identities.
7. The admitted peer retains the trusted public-key fingerprint and can be revalidated against current durable trust at explicit operation/lifecycle checkpoints.
8. One-to-one payload transfer additionally requires an explicit receiver application-authorization decision for the exact offered transfer before payload bytes are sent.
9. Transfer completion is accepted only when manifest, chunk, whole-payload, transfer-ID, size, and digest checks succeed.
10. Revocation or key replacement detected at a configured revalidation checkpoint closes the local peer; production scheduling and reconnect policy remain runtime responsibilities.

Pairing proof alone is not durable authorization. A valid bearer session alone is insufficient where trusted-device enforcement is configured. A valid TLS session also does not grant dataset, folder, destination-path, or payload-operation authorization by itself.

## Privacy and security requirements

- Privacy Shield governs consent, purpose, minimization, and data governance.
- Wardveil Security governs trust/security acceptance where applicable.
- Deleted application records use payload-free tombstones.
- Live records require payload and exact negotiated dataset/schema.
- Private keys, production bearer sessions, credentials, and private transfer contents must not be committed.
- Sync fails closed at enforced boundaries on invalid records, corrupt persistence, invalid proof, revoked/unknown trust, trust-store errors, inconsistent local keys, TLS identity/key mismatch, unsupported TLS, GC-SYNC identity mismatch, failed explicit trust revalidation, malformed transfer controls, unauthorized payload offers, bounded-frame violations, and payload integrity failures.
- Current JSON control and binary payload frames are bounded to 1 MiB. Binary chunk reads additionally enforce the exact expected chunk-size ceiling before allocation.
- Frame writes reject zero-progress short writes rather than silently emitting truncated protocol frames.
- Secure peers require TLS 1.3 and GoreeCloud Sync ALPN; raw helpers remain explicitly unauthenticated.
- Receiver application authorization occurs before payload transmission, and policy-denial details are not transmitted to the peer.
- Partial received bytes remain staging until the complete transfer verifies successfully.

## Storage and recovery

Current development persistence includes local JSON-backed replication/trust stores with explicit validation and private-file handling where implemented. The device private-key store requires a caller-supplied `KeyProtector`; no production secret-backend implementation is selected by this repository yet.

The current one-to-one payload receiver accepts an `io.Writer` staging destination only. It does not select, authorize, or publish a final filesystem path. Production file placement must add explicit path authorization, confinement, overwrite/conflict policy, atomic publication where appropriate, and recovery-aware behavior.

Production synchronized data must use approved GoreeCloud storage and least privilege. Live application databases, vault stores, certificate stores, backup repositories, and other consistency-sensitive state are not ordinary synchronization targets.

Sync does not replace Everkeep, GoreeCloud Backup, snapshots, or independent recovery points. A verified payload receipt proves integrity for that transfer, not recoverability or backup status.

## Platform-system integration

- **Privacy Shield:** source acceptance/minimization boundaries exist; production integration/evidence pending.
- **Wardveil Security:** source trust/acceptance boundaries exist; production integration/evidence pending.
- **Everkeep:** recovery authority is explicitly separate; product integration pending.
- **Glaze UI:** user-facing administration/client surfaces remain pending and must use current Stable Glaze UI when implemented.
- **GoreeCloud Mesh:** cross-application coordination remains explicit/versioned/least privilege; complete integration pending.
- **GoreeCloud Identity:** local device/session identity, pairing, durable trust, protected local key, secure-peer composition, explicit trust revalidation, and payload-transfer trust composition foundations exist; production account/runtime and Identity Center integration remain pending.

No platform-system name or interface label is evidence that production integration is complete.

## Functional and product requirements still pending

Major incomplete work includes:

- enabling secure peer and payload transfer in an approved runtime with discovery/address and listener/dial policy;
- production revalidation scheduling, background/session lifecycle behavior, and revocation-aware reconnect policy;
- broader transport/session/transfer replay and freshness controls beyond TLS, random transfer IDs, and one-time pairing challenges;
- production `KeyProtector` integration;
- production account/runtime wiring and user-facing trust administration/recovery/audit UX;
- LAN discovery and complete Nearby workflows;
- durable resumable transfer state and interrupted-transfer recovery;
- filesystem destination authorization, path confinement, overwrite/conflict policy, and atomic publication;
- persistent folder synchronization, watchers, ignore rules, direction modes, and deletion/conflict safety;
- complete multi-user folder authorization/administration;
- transfer progress, cancellation, prioritization, quotas, rate limits, and history;
- Share E2EE and recipient/expiry/revocation behavior;
- Glaze UI administration/client experiences;
- Android/Debian and other supported clients;
- monitoring, deployment, migration/rollback, production storage, performance acceptance, and operational runbooks;
- complete Glaze UI, Wardveil Security, Privacy Shield, Everkeep, GoreeCloud Mesh, and GoreeCloud Identity acceptance evidence.

## Development and validation

```bash
go test ./...
go vet ./...
go build ./cmd/goreecloud-sync
```

GitHub Actions also checks formatting. Passing CI proves only the exact validated source head and does not establish production deployment, release acceptance, or Stable qualification.

## Production-acceptance requirements

Before production-ready/Stable classification, evidence must cover the applicable final scope, including multi-user isolation/authorization, device enrollment/revocation/recovery, encrypted transport and secure-session lifecycle, revalidation/reconnect behavior, payload operation authorization, integrity, replay/freshness behavior, resource bounds, destination/path safety, conflicts/deletions, interrupted recovery, privacy-safe logs, Everkeep boundaries, migration/rollback, packaging, accessibility/Glaze UI, platform-system integration, exact release, and deployment validation.

## Current claim boundary

GoreeCloud Sync remains **pre-Stable**. Source existence, passing CI, secure TLS admission, durable trust, explicit revalidation, development record replication, or the bounded authenticated one-to-one payload stream must not be represented as a completed production Sync, Nearby, or Share product.
