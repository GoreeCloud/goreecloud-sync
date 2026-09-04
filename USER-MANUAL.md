# GoreeCloud Sync User Manual

## Current status

GoreeCloud Sync is under active development and is pre-Stable. Current source includes first-party application-record replication foundations, device pairing/trust, TLS 1.3 authenticated peer transport, explicit current-trust revalidation and operation guards, transfer manifests/integrity verification, and a bounded authenticated one-to-one file/text payload-transfer primitive. The default development service is not a production synchronization deployment.

## Run the development service

From the repository:

```bash
go test ./...
go vet ./...
go build ./cmd/goreecloud-sync
go run ./cmd/goreecloud-sync serve
```

The default listener is `127.0.0.1:8787`.

Development status endpoints:

```text
GET /healthz
GET /api/v1/status
```

The default `serve` command does not automatically enable secure peer listeners, peer discovery, payload-transfer orchestration, production account authority, Nearby, Share, or full folder synchronization.

## Device trust model

A device is not trusted merely because it is reachable on the network.

Current source separates the process into several steps:

1. A device has an Ed25519 identity.
2. Pairing proof must satisfy an exact short-lived one-time challenge.
3. A verified pairing may be explicitly authorized into the selected account's durable trusted-device store.
4. Secure peer admission resolves the currently trusted remote device and exact public key before TLS 1.3 is established.
5. The TLS session proves possession of the pinned key and binds the GC-SYNC device identity to that authenticated peer.
6. The factory binds the trusted public-key fingerprint to the admitted connection for later current-trust checks.
7. A payload transfer still requires a separate receiver application-authorization decision before any file/text bytes are sent.

Pairing proof, network reachability, a TLS session, or an authenticated application session alone does not grant every dataset, folder, destination, or transfer permission.

## Secure peer revalidation

Higher-level runtime code can explicitly call the secure-peer trust revalidation checkpoint before continuing to use an established session.

Revalidation checks the exact:

- account;
- authenticated remote device ID;
- public-key fingerprint bound at secure admission.

against the current trusted-device store.

If the trusted device was revoked, replaced, removed, became corrupt, or the trust store cannot be read safely, revalidation fails closed and closes the local peer connection.

Source provides `RunWithCurrentTrust` for one operation boundary and `RunOperationSequenceWithCurrentTrust` for a validated sequence of up to 64 operations. Both reuse the same current-trust check rather than creating a separate authorization rule.

Payload-transfer application composition also revalidates before transfer start, before sender source reads, before receiver staging writes, and before returning verified success.

### Important lifecycle limitation

Revalidation remains explicit and checkpoint-based, not background/instantaneous. These controls reduce the chance that higher-level operations silently reuse stale trust, but the source does not claim that every established connection closes at the exact moment a trust record changes, nor can a checkpoint retroactively stop bytes already written after it succeeded.

An approved production runtime must choose complete session/reconnect policy and any additional periodic or event-driven revalidation behavior required for long-lived sessions.

## One-to-one file and text payload foundation

Current source can move one already-manifested file or logical text payload over an authenticated GoreeCloud Sync peer when the peer has a durable trusted-device fingerprint bound by the application trust layer.

The Development flow is:

1. The sender has a validated manifest describing the payload and its chunk/whole-payload SHA-256 digests.
2. A fresh random transfer ID is assigned.
3. The sender offers the file/text transfer.
4. Receiver application logic explicitly accepts or rejects the offer.
5. No payload bytes are sent when the receiver rejects it.
6. After acceptance, the sender verifies each source chunk before sending it through the authenticated encrypted peer.
7. The receiver verifies each chunk before writing it to the caller's staging destination.
8. The sender verifies the complete source and sends a completion record.
9. The receiver verifies the complete staged payload and returns a receipt marked `verified` only when transfer ID, size, and hash all match.

Protocol, framing, stream, staging-write, or integrity failures close the peer fail-closed instead of silently continuing on a potentially desynchronized stream.

### Staging requirement

The receive primitive writes to a caller-provided staging destination. Do not treat partially written bytes as a completed file. The caller must publish, rename, or otherwise commit staged content only after the receive operation returns verified success.

The current source does not yet select or authorize a final filesystem path, prevent path/symlink escape at a final destination, define overwrite/conflict behavior, persist resume progress, expose user-facing Nearby controls, or wire this primitive into the default `serve` runtime.

A verified transfer proves payload integrity for this transfer only. It is not backup evidence and does not establish recoverability.

## First-party application-record Sync

Source handlers exist for approved Browser, Search, and Bookmarks datasets when the service is explicitly constructed with the required ingestion and authenticated-peer dependencies.

Current datasets include:

```text
search.history
bookmarks.items
browser.tabs
browser.history
```

Records use exact negotiated dataset/schema validation, bounded identifiers, deterministic ordering/conflict behavior, and payload-free deletion tombstones.

This source foundation is not a general folder synchronization engine and is not production deployment evidence.

## Retrieval limits

Authenticated retrieval currently uses record-ID cursor pagination:

- default page size: 256;
- maximum page size: 1,024;
- record IDs/cursors: maximum 512 bytes.

Persisted state is validated before records are returned so off-page corruption cannot silently pass through a requested page.

## Privacy and security

Privacy Shield remains authoritative for purpose, consent, minimization, and data-governance decisions. Wardveil Security remains authoritative for applicable GoreeCloud trust/security acceptance. Sync does not invent those statuses itself.

Device private keys and production credentials must not be committed to the repository. The local device-key store requires a caller-provided `KeyProtector`; a production key protector has not yet been selected/accepted by this repository.

Receiver policy rejection details are not sent to the remote peer through the current payload decision record. The current random transfer ID and ordered stream contract are also not presented as a complete production anti-replay/freshness policy; broader controls remain pending.

## Backup and recovery

Synchronization is not backup. Everkeep remains the GoreeCloud continuity, preservation, backup, and recovery authority. A synchronized replica or verified received payload should not be treated as an independent recovery point unless Everkeep/backup acceptance explicitly establishes that property.

## Features not yet production-ready

- production secure listener/dial and payload-transfer orchestration;
- approved discovery and address selection;
- complete operation/session revalidation cadence and revocation-aware reconnect policy;
- production transfer replay/freshness controls;
- durable transfer resume persistence and interrupted-transfer recovery;
- destination path authorization/confinement, overwrite/conflict behavior, and atomic final publication;
- transfer progress, cancellation, prioritization, rate limits, quotas, and history;
- production GoreeCloud Identity/account integration;
- user-facing device approval/revocation/recovery administration;
- complete folder synchronization;
- LAN Nearby discovery and user-facing transfer workflows;
- Share end-to-end encrypted delivery;
- full Glaze UI administration and client surfaces;
- supported Android/Debian client delivery;
- complete Privacy Shield, Wardveil Security, Everkeep, Mesh, and Identity production acceptance;
- production deployment and Stable release.

## Troubleshooting

If secure admission fails after a device was revoked, that is expected fail-closed behavior. Re-authorize the device only through the approved pairing/trust process rather than editing the trusted-device file manually.

If an established peer is closed by current-trust revalidation or a payload transfer stops between chunks after revocation, inspect current account-scoped trust state. A changed key fingerprint is treated as a different trust state and must not be silently accepted.

If a payload receive fails after writing some staging bytes, discard or otherwise handle that uncommitted staging data according to the future approved destination/recovery workflow; do not publish it as a verified file.

If the default development service does not expose full Sync routes or peer networking, that is expected: those capabilities require explicit dependency/runtime composition and are not enabled by the base `serve` command.
