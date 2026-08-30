# GoreeCloud Sync Specifications

## Status

**Lifecycle:** Active Development — pre-Stable.

GoreeCloud Sync is original GoreeCloud-owned software for private synchronization and secure transfer. Current source implements development foundations for first-party application-record replication, peer transport, device identity, pairing proof/challenge handling, durable trusted-device authorization, and fail-closed runtime trust checks. It is not a complete production synchronization product and is not approved as Stable.

## Purpose and scope

GoreeCloud Sync is intended to provide three related product modes while keeping their authorization and security boundaries explicit:

- **Sync** — persistent replication of explicitly approved application datasets and, later, approved folders.
- **Nearby** — direct device-to-device transfer over local or approved private-network paths.
- **Share** — temporary encrypted delivery to recipients, including expiring links where implemented later.

Synchronization does not establish backup authority. Everkeep remains the GoreeCloud continuity, backup, preservation, and recovery authority.

## Implemented source foundations

Current merged source includes:

- a Go service and CLI shell with loopback-safe development defaults, health/status endpoints, and graceful shutdown;
- pre-stabilization `GC-SYNC/1` capability handshakes, bounded control framing, and context-bound TCP peer helpers;
- Ed25519 device identity, key fingerprinting, and signed pairing proofs;
- cryptographically random, short-lived one-time pairing challenges with expiry and replay rejection;
- `VerifiedPairing` output only after proof verification and exact challenge consumption;
- a durable account-scoped trusted-device registry with explicit revocation, private filesystem permissions, atomic publication, stored-key/fingerprint consistency checks, and refusal to silently replace an active device key;
- a trusted peer resolver that composes existing authenticated peer resolution with exact account/device/key-fingerprint authorization and fails closed on unknown, revoked, mismatched, or trust-store-error state;
- first-party dataset capability negotiation and replication handlers for `search.history`, `bookmarks.items`, `browser.tabs`, and `browser.history`;
- transport-neutral record envelopes, deterministic conflict resolution, payload-free tombstones, record-bound proof foundations, replay/high-water foundations, observation receipts, and tombstone-convergence controls;
- authenticated bounded retrieval ordered by record ID, with a 512-byte record/cursor bound, default page size 256, maximum page size 1,024, full persisted-store validation, and page-bounded candidate retention.

These are source and development contracts. The default development `serve` command does not by itself wire the complete replication/trust runtime, production account authority, or encrypted peer-session establishment.

## Architecture

Primary repository areas are:

- `cmd/goreecloud-sync/` — CLI and service entry point.
- `internal/app/` — HTTP service composition, ingestion/retrieval handlers, authenticated peer resolution, and trusted-peer enforcement.
- `internal/identity/` — device keys, pairing proofs, one-time challenges, trusted-device authorization, fingerprints, and related validation.
- `internal/session/` — authenticated peer/session models and session-bound identity state.
- `internal/transport/` — pre-stabilization framing, handshake, and peer-stream helpers.
- `internal/datasets/` — dataset capabilities, record envelopes, validation, and conflict behavior.
- `internal/replication/` — persistence, ingestion state, retrieval support, replay/receipt/tombstone foundations.
- `internal/policy/` — Privacy Shield and Wardveil-facing acceptance decision boundaries.
- `protocol/` — pre-stabilization protocol records and interoperability fixtures.
- `docs/` — architecture, security, threat-model, and engineering documentation.

Application semantics remain owned by the originating GoreeCloud application. Sync coordinates authorized replication and must not become an implicit authority over Browser, Search, Bookmarks, or other application data.

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
5. Revocation removes that trust and later checks fail closed.

Pairing proof alone is not durable authorization. A valid bearer/authenticated session alone is also insufficient when trusted-device enforcement is configured.

Authenticated encrypted peer-session establishment is **not yet implemented as a completed product boundary**. Future transport encryption must use established, reviewed cryptographic libraries/protocols rather than custom cryptography.

## Privacy and security requirements

- Privacy Shield governs consent, purpose, minimization, and data-governance decisions where applicable.
- Wardveil Security governs trust/security acceptance decisions where applicable.
- Deleted application records use payload-free tombstones; deleted content must not be retained merely to communicate deletion.
- Live records require application payload and must match the exact negotiated dataset/schema.
- Private keys, production bearer sessions, reusable credentials, and private transfer contents must not be committed to the repository.
- Sync must fail closed on invalid negotiated records, corrupt persisted state, invalid proof shape, revoked/unknown device trust, and trust-store errors at enforced boundaries.
- Current control frames are bounded to 1 MiB and reject malformed length, truncation, unknown fields, and trailing values.

## Storage and recovery

Current development persistence includes local JSON-backed replication/trust stores with explicit validation and private-file handling where implemented. These stores are source foundations, not the final production multi-user storage architecture.

Production synchronized data must use approved GoreeCloud storage and least-privilege access. Live application databases, vault stores, certificate stores, backup repositories, and other consistency-sensitive state are not ordinary synchronization targets.

Sync does not replace Everkeep, GoreeCloud Backup, snapshots, or independent recovery points. A synchronized copy is not automatically a recoverable backup.

## Platform-system integration

Applicable GoreeCloud platform responsibilities are mandatory but are at different implementation stages:

- **Privacy Shield:** source acceptance/minimization boundaries exist; complete production integration and evidence remain pending.
- **Wardveil Security:** source trust/acceptance boundaries exist; complete production integration and evidence remain pending.
- **Everkeep:** recovery authority is explicitly separated from Sync; protection-awareness/product integration remains pending.
- **Glaze UI:** user-facing administrative and client surfaces remain pending and must use the latest applicable Stable Glaze UI contract when implemented.
- **GoreeCloud Mesh:** cross-application coordination must remain explicit, versioned, and least privilege; complete Mesh integration remains pending.
- **GoreeCloud Identity:** device/session identity foundations exist locally in Sync, but production account/runtime integration and broader Identity Center integration remain pending.

No platform-system name or interface label may be treated as evidence that production integration is complete.

## Functional and product requirements still pending

Major incomplete work includes:

- reviewed authenticated encryption for peer sessions and transport/session replay protection beyond the implemented one-time pairing challenge;
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
- trusted-device enrollment, revocation, and recovery behavior;
- authenticated encrypted transport and replay resistance;
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

The repository is actively developing substantial Sync foundations, but GoreeCloud Sync is **pre-Stable**. Source existence, successful CI, pairing/trust foundations, or development record replication must not be represented as a completed production synchronization, Nearby, or Share product.