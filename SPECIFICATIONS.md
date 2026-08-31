# GoreeCloud Sync Specifications

## Status

**Lifecycle:** Active Development — pre-Stable.

GoreeCloud Sync is original GoreeCloud-owned software for private synchronization and secure transfer. Current source implements development foundations for first-party application-record replication, raw and TLS-authenticated peer transport, device identity, pairing proof/challenge handling, durable trusted-device authorization, fail-closed runtime trust checks, and source-level composition of current durable trust with protected local identity for secure peer admission. It is not a complete production synchronization product and is not approved as Stable.

## Purpose and scope

GoreeCloud Sync is intended to provide three related product modes while keeping their authorization and security boundaries explicit:

- **Sync** — persistent replication of explicitly approved application datasets and, later, approved folders.
- **Nearby** — direct device-to-device transfer over local or approved private-network paths.
- **Share** — temporary encrypted delivery to recipients, including expiring links where implemented later.

Synchronization does not establish backup authority. Everkeep remains the GoreeCloud continuity, backup, preservation, and recovery authority.

## Implemented source foundations

Current merged source includes:

- a Go service and CLI shell with loopback-safe development defaults, health/status endpoints, and graceful shutdown;
- pre-stabilization `GC-SYNC/1` capability handshakes, bounded control framing, and context-bound raw TCP peer helpers;
- a separate TLS 1.3-only mutually authenticated secure-peer wrapper for already-trusted devices, using Go's reviewed `crypto/tls` and `crypto/x509` implementations;
- short-lived self-signed Ed25519 certificates used as TLS key-possession carriers, with exact expected device-ID/raw-public-key pinning, `goreecloud-sync/1` ALPN binding, bounded TLS handshakes, session-ticket disabling, mutual acceptance confirmation, and GC-SYNC capability-handshake identity binding;
- Ed25519 device identity, key fingerprinting, and signed pairing proofs;
- cryptographically random, short-lived one-time pairing challenges with expiry and replay rejection;
- `VerifiedPairing` output only after proof verification and exact challenge consumption;
- a hardened local device-key store that requires canonical device IDs, internally consistent Ed25519 key pairs, validated stored public-key/fingerprint metadata, owner-only storage permissions, protected private-key opening through a `KeyProtector`, and non-mutating chronology validation before rotation publication;
- a durable account-scoped trusted-device registry with explicit revocation, private filesystem permissions, atomic publication, stored-key/fingerprint consistency checks, and refusal to silently replace an active device key;
- a trusted peer resolver that composes existing authenticated peer resolution with exact account/device/key-fingerprint authorization and fails closed on unknown, revoked, mismatched, or trust-store-error state;
- an application-layer secure-peer factory that resolves a current non-revoked account-scoped trusted remote device, obtains its validated Ed25519 public key, opens the hardened local protected device identity, and passes those exact identities into the existing TLS 1.3 dial/accept primitives;
- first-party dataset capability negotiation and replication handlers for `search.history`, `bookmarks.items`, `browser.tabs`, and `browser.history`;
- transport-neutral record envelopes, deterministic conflict resolution, payload-free tombstones, record-bound proof foundations, replay/high-water foundations, observation receipts, and tombstone-convergence controls;
- authenticated bounded retrieval ordered by record ID, with a 512-byte record/cursor bound, default page size 256, maximum page size 1,024, full persisted-store validation, and page-bounded candidate retention.

These are source and development contracts. The default development `serve` command does not by itself enable the secure-peer factory, peer listeners or dials, discovery/address policy, production account authority, complete replication/trust runtime, revocation-aware reconnect behavior, or full secure-session lifecycle management.

## Architecture

Primary repository areas are:

- `cmd/goreecloud-sync/` — CLI and service entry point.
- `internal/app/` — HTTP service composition, ingestion/retrieval handlers, authenticated peer resolution, trusted-peer enforcement, and secure-peer identity/transport composition.
- `internal/identity/` — device keys, pairing proofs, one-time challenges, trusted-device authorization, fingerprints, and related validation.
- `internal/session/` — authenticated peer/session models and session-bound identity state.
- `internal/transport/` — pre-stabilization framing/handshake, raw TCP peer helpers, and TLS 1.3 authenticated secure-peer helpers.
- `internal/datasets/` — dataset capabilities, record envelopes, validation, and conflict behavior.
- `internal/replication/` — persistence, ingestion state, retrieval support, replay/receipt/tombstone foundations.
- `internal/policy/` — Privacy Shield and Wardveil-facing acceptance decision boundaries.
- `protocol/` — pre-stabilization protocol records and interoperability fixtures.
- `docs/` — architecture, security, threat-model, and engineering documentation.

Application semantics remain owned by the originating GoreeCloud application. Sync coordinates authorized replication and must not become an implicit authority over Browser, Search, Bookmarks, or other application data.

## Peer transport boundary

Raw `DialPeer` and `AcceptPeer` establish/wrap context-bound TCP streams and intentionally assign no trust. They remain lower-level primitives.

`DialSecurePeer` and `AcceptSecurePeer` add a TLS 1.3-only authentication/encryption layer for a peer whose exact device identity is already trusted by higher-level authorization logic. The current secure-peer contract:

- requires a canonical local device ID and an in-memory Ed25519 private key;
- requires the expected remote canonical device ID and exact raw Ed25519 public key;
- requires mutual TLS certificate possession;
- uses short-lived self-signed certificates rather than public Web PKI as the trust root;
- pins the expected remote device ID and public key in `VerifyConnection`;
- requires the GoreeCloud Sync ALPN value;
- uses bounded handshake time;
- requires a TLS-protected accepting-side confirmation before the dialer reports the secure peer established;
- binds later GC-SYNC capability-handshake device IDs to the identities already authenticated by TLS;
- does not persist private key material.

`SecurePeerFactory` is the current source composition boundary above that transport. For each explicit dial/accept call it resolves the requested remote device from current account-scoped durable trust before opening the local private key, rejects revoked/unknown/cross-account/corrupt trust state, loads the hardened local identity through its `KeyProtector`, and then invokes the existing TLS primitive with the resolved exact identities. An accepted stream is closed if identity resolution fails before TLS admission.

The caller remains responsible for selecting the account context, remote device ID, address/listener, and connection lifecycle. The default service does not automatically enable this factory. A network route, self-signed certificate, factory object, or successful raw TCP connection alone never establishes dataset/folder/application authorization.

## Service interfaces

The base development service listens on `127.0.0.1:8787` by default and exposes:

- `GET /healthz`
- `GET /api/v1/status`

When explicitly constructed with the required replication ingestor and authenticated peer resolver, source handlers exist for:

- `POST` and `GET /api/v1/sync/search/history`
- `POST` and `GET /api/v1/sync/bookmarks/items`
- `POST` and `GET /api/v1/sync/browser/tabs`
- `POST` and `GET /api/v1/sync/browser/history`

Retrieval uses `limit` and exclusive `after` cursor parameters. A record identifier or cursor above 512 bytes is rejected. Persisted state is validated against the negotiated capability before records are emitted.

## Authentication, authorization, and trust

A network connection is not application authorization.

Current source separates several trust steps:

1. A device proves possession of its Ed25519 key with a signed pairing proof.
2. The proof must consume the exact short-lived one-time challenge; expired or reused challenges are rejected.
3. Explicit application logic may authorize the resulting `VerifiedPairing` into an account-scoped durable trusted-device store.
4. Authenticated peer resolution must still match the account-scoped trusted device ID and key fingerprint before dataset ingestion or retrieval can proceed when the trusted resolver is wired.
5. For direct secure peer transport, `SecurePeerFactory` can resolve the requested current non-revoked remote device from the selected account's durable trust state, validate its stored public key/fingerprint, load the hardened local protected device identity, and pass those exact values to the TLS 1.3 wrapper.
6. The TLS layer proves exact key possession/identity and requires accepting-side confirmation before returning the dialed secure peer; the subsequent GC-SYNC capability handshake must claim the same identities.
7. Revocation removes durable trust for future factory resolution. Revocation of already-established sessions, reconnect policy, and production lifecycle orchestration remain separate runtime responsibilities.

Pairing proof alone is not durable authorization. A valid bearer/authenticated HTTP session alone is insufficient when trusted-device enforcement is configured. A valid TLS cryptographic session also does not grant dataset/folder/application authorization by itself.

## Privacy and security requirements

- Privacy Shield governs consent, purpose, minimization, and data-governance decisions where applicable.
- Wardveil Security governs trust/security acceptance decisions where applicable.
- Deleted application records use payload-free tombstones; deleted content must not be retained merely to communicate deletion.
- Live records require application payload and must match the exact negotiated dataset/schema.
- Private keys, production bearer sessions, reusable credentials, and private transfer contents must not be committed to the repository.
- Sync must fail closed on invalid negotiated records, corrupt persisted state, invalid proof shape, revoked/unknown device trust, trust-store errors at enforced boundaries, inconsistent local device-key state, TLS device/key mismatches, unsupported TLS versions, and GC-SYNC identity mismatches on secure peers.
- Current control frames are bounded to 1 MiB and reject malformed length, truncation, unknown fields, and trailing values.
- Current secure peer sessions require TLS 1.3 and the GoreeCloud Sync ALPN; raw peer helpers remain explicitly unauthenticated.

## Storage and recovery

Current development persistence includes local JSON-backed replication/trust stores with explicit validation and private-file handling where implemented. The device private-key store requires a caller-supplied `KeyProtector`; the repository does not select or ship a production secret-backend implementation. These stores are source foundations, not the final production multi-user storage architecture.

Production synchronized data must use approved GoreeCloud storage and least-privilege access. Live application databases, vault stores, certificate stores, backup repositories, and other consistency-sensitive state are not ordinary synchronization targets.

Sync does not replace Everkeep, GoreeCloud Backup, snapshots, or independent recovery points. A synchronized copy is not automatically a recoverable backup.

## Platform-system integration

Applicable GoreeCloud platform responsibilities are mandatory but are at different implementation stages:

- **Privacy Shield:** source acceptance/minimization boundaries exist; complete production integration and evidence remain pending.
- **Wardveil Security:** source trust/acceptance boundaries exist; complete production integration and evidence remain pending.
- **Everkeep:** recovery authority is explicitly separated from Sync; protection-awareness/product integration remains pending.
- **Glaze UI:** user-facing administrative and client surfaces remain pending and must use the latest applicable Stable Glaze UI contract when implemented.
- **GoreeCloud Mesh:** cross-application coordination must remain explicit, versioned, and least privilege; complete Mesh integration remains pending.
- **GoreeCloud Identity:** device/session identity, pairing, durable trust, protected local key, trusted-device resolution, and direct secure-peer composition primitives exist locally in Sync, but production account/runtime integration and broader Identity Center integration remain pending.

No platform-system name or interface label may be treated as evidence that production integration is complete.

## Functional and product requirements still pending

Major incomplete work includes:

- enabling the secure-peer factory in an approved runtime with discovery/address and listener/dial policy, secure-session lifecycle management, revocation-aware reconnect behavior, and broader transport/session replay/freshness controls beyond TLS and the implemented one-time pairing challenge;
- selecting and validating a concrete production `KeyProtector`/secret-backend integration for supported deployment targets;
- production account/runtime wiring and user-facing pairing approval, trust management, revocation, recovery, and audit UX;
- LAN discovery and completed Nearby flows;
- durable resumable file/folder synchronization, filesystem watching, direction modes, ignore rules, and production conflict/deletion workflows;
- complete multi-user account, folder-authorization, isolation, and administration surfaces;
- Share end-to-end encryption, expiration, revocation, recipient controls, and relay/storage behavior;
- Glaze UI administration and client experiences;
- native Android client, Debian packaging, and other supported client delivery;
- monitoring, deployment, migration/rollback, production storage, performance acceptance, and operational runbooks;
- complete Glaze UI, Wardveil Security, Privacy Shield, Everkeep, GoreeCloud Mesh, and GoreeCloud Identity acceptance evidence appropriate to the final scope.

## Development and validation

Current repository validation includes:

```bash
go test ./...
go vet ./...
go build ./cmd/goreecloud-sync
```

GitHub Actions also checks formatting before test, vet, and build validation. Passing CI proves the validated source head only; it does not establish production deployment, release acceptance, or Stable qualification.

## Production-acceptance requirements

Before GoreeCloud Sync can be classified production-ready or Stable, evidence must cover the implemented scope, including as applicable:

- multi-user isolation and authorization;
- trusted-device enrollment, revocation, recovery, and connection-admission behavior;
- authenticated encrypted transport, downgrade resistance, replay/freshness behavior, and secure-session lifecycle;
- transfer and record integrity;
- resource bounds and abuse resistance;
- conflict and deletion safety;
- interrupted-transfer recovery;
- privacy-safe logging and data minimization;
- backup/recovery compatibility and Everkeep boundaries;
- migration and rollback;
- packaging and supported-platform verification;
- accessibility and Glaze UI conformance;
- Wardveil Security, Privacy Shield, Everkeep, GoreeCloud Mesh, and GoreeCloud Identity integration evidence;
- exact-release and deployment validation.

## Current claim boundary

The repository is actively developing substantial Sync foundations, including hardened local device-key integrity, durable account-scoped trust, a TLS 1.3 authenticated peer primitive, and a source-level factory that composes current trust/local identity into secure dial/accept calls, but GoreeCloud Sync is **pre-Stable**. Source existence, successful CI, pairing/trust/TLS foundations, secure-peer composition, or development record replication must not be represented as a completed production synchronization, Nearby, or Share product.
