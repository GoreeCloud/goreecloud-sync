# GoreeCloud Sync Specifications

## Status

**Lifecycle:** Active Development — pre-Stable.

GoreeCloud Sync is original GoreeCloud-owned software for private synchronization and secure transfer. Current source implements development foundations for first-party application-record replication, raw and TLS-authenticated peer transport, device identity, pairing proof/challenge handling, durable trusted-device authorization, fail-closed runtime trust checks, and source-level composition/revalidation of current durable trust with protected local identity for secure peer sessions. It is not a complete production synchronization product and is not approved as Stable.

## Purpose and scope

GoreeCloud Sync is intended to provide three related product modes while keeping their authorization and security boundaries explicit:

- **Sync** — persistent replication of explicitly approved application datasets and, later, approved folders.
- **Nearby** — direct device-to-device transfer over local or approved private-network paths.
- **Share** — temporary encrypted delivery to recipients, including expiring links where implemented later.

Synchronization does not establish backup authority. Everkeep remains the GoreeCloud continuity, backup, preservation, and recovery authority.

## Implemented source foundations

Current source includes:

- a Go service and CLI shell with loopback-safe development defaults, health/status endpoints, and graceful shutdown;
- pre-stabilization `GC-SYNC/1` capability handshakes, bounded control framing, and context-bound raw TCP peer helpers;
- a separate TLS 1.3-only mutually authenticated secure-peer wrapper for already-trusted devices, using Go's reviewed `crypto/tls` and `crypto/x509` implementations;
- short-lived self-signed Ed25519 certificates used as TLS key-possession carriers, with exact expected device-ID/raw-public-key pinning, `goreecloud-sync/1` ALPN binding, bounded TLS handshakes, session-ticket disabling, mutual acceptance confirmation, and GC-SYNC capability-handshake identity binding;
- Ed25519 device identity, key fingerprinting, and signed pairing proofs;
- cryptographically random, short-lived one-time pairing challenges with expiry and replay rejection;
- `VerifiedPairing` output only after proof verification and exact challenge consumption;
- a hardened local device-key store with canonical IDs, consistent Ed25519 material, protected private-key opening, owner-only persistence, and atomic rotation;
- a durable account-scoped trusted-device registry with explicit revocation, atomic/private persistence, stored-key/fingerprint validation, and active-key replacement protection;
- a trusted peer resolver requiring the exact account/device/key fingerprint to remain authorized;
- an application-layer secure-peer factory that resolves current non-revoked remote trust before opening the local key, establishes the existing TLS 1.3 session, binds the fingerprint of the admitted trusted public key to the resulting peer, and provides an explicit current-trust revalidation checkpoint;
- first-party dataset capability negotiation and replication handlers for `search.history`, `bookmarks.items`, `browser.tabs`, and `browser.history`;
- transport-neutral record envelopes, deterministic conflict resolution, payload-free tombstones, record-bound proof foundations, replay/high-water foundations, observation receipts, and tombstone-convergence controls;
- authenticated bounded retrieval ordered by record ID, with a 512-byte record/cursor bound, default page size 256, maximum page size 1,024, full persisted-store validation, and page-bounded candidate retention.

These are source and development contracts. The default development `serve` command does not by itself enable secure peer listeners/dials, discovery/address policy, production account authority, revalidation scheduling, revocation-aware reconnect policy, or full secure-session lifecycle management.

## Architecture

Primary repository areas are:

- `cmd/goreecloud-sync/` — CLI and service entry point.
- `internal/app/` — HTTP composition, ingestion/retrieval, authenticated peer resolution, trust enforcement, secure-peer identity composition, and explicit trust revalidation.
- `internal/identity/` — device keys, pairing proofs, one-time challenges, trusted-device authorization, fingerprints, and validation.
- `internal/session/` — authenticated peer/session models and session-bound identity state.
- `internal/transport/` — pre-stabilization framing/handshake, raw TCP helpers, and TLS 1.3 authenticated secure-peer helpers.
- `internal/datasets/` — capabilities, record envelopes, validation, and conflict behavior.
- `internal/replication/` — persistence, ingestion state, retrieval, replay/receipt/tombstone foundations.
- `internal/policy/` — Privacy Shield and Wardveil-facing acceptance boundaries.
- `protocol/` — pre-stabilization protocol records and interoperability fixtures.
- `docs/` — architecture, security, threat-model, and engineering documentation.

Application semantics remain owned by the originating GoreeCloud application. Sync coordinates authorized replication and must not become an implicit authority over Browser, Search, Bookmarks, or other application data.

## Peer transport boundary

Raw `DialPeer` and `AcceptPeer` establish/wrap context-bound TCP streams and intentionally assign no trust.

`DialSecurePeer` and `AcceptSecurePeer` add TLS 1.3 authentication/encryption for a peer whose exact identity is already trusted. The transport requires exact device identity and Ed25519 key pinning, mutual certificate possession, GoreeCloud Sync ALPN, bounded handshake time, accepting-side confirmation, and later GC-SYNC identity consistency.

`SecurePeerFactory` is the application trust composition layer. For each explicit dial/accept call it resolves the requested remote device from current account-scoped durable trust before opening the local private key, rejects revoked/unknown/cross-account/corrupt trust, and invokes the TLS primitive with those exact identities. After secure admission it binds the fingerprint of that exact validated remote public key to `PeerConn`.

`SecurePeerFactory.RevalidatePeer` is an explicit current-authorization checkpoint. It compares the authenticated peer device ID and bound key fingerprint with the current account-scoped `TrustedDeviceStore`. A revoked, replaced, missing, corrupt, or unreadable trust record fails closed and closes the local peer.

This checkpoint is not asynchronous revocation. Runtime orchestration remains responsible for choosing when to revalidate long-lived or reused sessions, and for reconnect/listener lifecycle policy. A trust change does not imply every already-established connection is automatically terminated at the same instant unless a future accepted runtime implements that behavior.

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
8. Revocation or key replacement detected at revalidation closes the local peer; scheduling those checkpoints and reconnect policy remain runtime responsibilities.

Pairing proof alone is not durable authorization. A valid bearer session alone is insufficient where trusted-device enforcement is configured. A valid TLS session also does not grant dataset/folder/application authorization by itself.

## Privacy and security requirements

- Privacy Shield governs consent, purpose, minimization, and data governance.
- Wardveil Security governs trust/security acceptance where applicable.
- Deleted application records use payload-free tombstones.
- Live records require payload and exact negotiated dataset/schema.
- Private keys, production bearer sessions, credentials, and private transfer contents must not be committed.
- Sync fails closed at enforced boundaries on invalid records, corrupt persistence, invalid proof, revoked/unknown trust, trust-store errors, inconsistent local keys, TLS identity/key mismatch, unsupported TLS, GC-SYNC identity mismatch, and failed explicit session revalidation.
- Current control frames are bounded to 1 MiB and reject malformed length, truncation, unknown fields, and trailing values.
- Secure peers require TLS 1.3 and GoreeCloud Sync ALPN; raw helpers remain explicitly unauthenticated.

## Storage and recovery

Current development persistence includes local JSON-backed replication/trust stores with explicit validation and private-file handling where implemented. The device private-key store requires a caller-supplied `KeyProtector`; no production secret-backend implementation is selected by this repository yet.

Production synchronized data must use approved GoreeCloud storage and least privilege. Live application databases, vault stores, certificate stores, backup repositories, and other consistency-sensitive state are not ordinary synchronization targets.

Sync does not replace Everkeep, GoreeCloud Backup, snapshots, or independent recovery points.

## Platform-system integration

- **Privacy Shield:** source acceptance/minimization boundaries exist; production integration/evidence pending.
- **Wardveil Security:** source trust/acceptance boundaries exist; production integration/evidence pending.
- **Everkeep:** recovery authority is explicitly separate; product integration pending.
- **Glaze UI:** user-facing administration/client surfaces remain pending and must use current Stable Glaze UI when implemented.
- **GoreeCloud Mesh:** cross-application coordination remains explicit/versioned/least privilege; complete integration pending.
- **GoreeCloud Identity:** local device/session identity, pairing, durable trust, protected local key, secure-peer composition, and explicit trust revalidation foundations exist; production account/runtime and Identity Center integration remain pending.

No platform-system name or interface label is evidence that production integration is complete.

## Functional and product requirements still pending

Major incomplete work includes:

- enabling secure peer transport in an approved runtime with discovery/address and listener/dial policy;
- selecting and enforcing revalidation checkpoints for long-lived/reused sessions and implementing revocation-aware reconnect lifecycle policy;
- broader transport/session replay and freshness controls beyond TLS and one-time pairing challenges;
- production `KeyProtector` integration;
- production account/runtime wiring and user-facing trust administration/recovery/audit UX;
- LAN discovery and Nearby;
- durable resumable file/folder synchronization and filesystem policies;
- complete multi-user folder authorization/administration;
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

Before production-ready/Stable classification, evidence must cover the applicable final scope, including multi-user isolation/authorization, device enrollment/revocation/recovery, encrypted transport and secure-session lifecycle, revalidation/reconnect behavior, integrity, resource bounds, conflicts/deletions, interrupted recovery, privacy-safe logs, Everkeep boundaries, migration/rollback, packaging, accessibility/Glaze UI, platform-system integration, exact release, and deployment validation.

## Current claim boundary

GoreeCloud Sync remains **pre-Stable**. Source existence, passing CI, secure TLS admission, durable trust, explicit revalidation, or development record replication must not be represented as a completed production Sync, Nearby, or Share product.
