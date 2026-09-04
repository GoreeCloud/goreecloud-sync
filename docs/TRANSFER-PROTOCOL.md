# GoreeCloud Sync Transfer Protocol

## Status

Development foundation only. `GC-SYNC/1` and the current payload-transfer records are internal pre-stabilization contracts and are not approved Stable wire protocols or production compatibility promises.

## Purpose

This document records the current transfer-engine and direct-peer transport foundations being developed for GoreeCloud Sync.

## Current source foundation

Current source provides:

- device Ed25519 key-generation and public-key fingerprinting primitives;
- protected device-key storage abstractions;
- pairing proof/challenge and durable trusted-device foundations;
- chunk and transfer-session data models;
- SHA-256 content-digest helpers;
- versioned single-file transfer manifests built by streaming the source payload rather than materializing the complete file in memory;
- deterministic ordered chunk metadata with canonical lowercase SHA-256 hashes;
- structural manifest validation for declared size, chunk count/order, current 1 MiB chunk bounds, and digest encoding;
- per-chunk verification plus final whole-payload verification that fails closed on corruption, truncation, or undeclared trailing bytes;
- raw context-bound TCP peer helpers that do not imply trust;
- a TLS 1.3 authenticated direct-peer primitive for already-trusted device identities;
- mutual Ed25519 key-possession proof with exact expected device-ID/raw-key pinning;
- GoreeCloud Sync ALPN and GC-SYNC capability-handshake identity binding;
- bounded TLS handshake time and session-ticket disabling in the current secure wrapper;
- bounded binary payload frames with the same 1 MiB global frame ceiling and a stricter caller-supplied expected-chunk allocation bound;
- versioned one-to-one file/text payload offers, explicit receiver decisions, sender completion records, and verified receiver receipts;
- random 128-bit transfer identifiers encoded as canonical lowercase hexadecimal;
- a payload stream that requires a TLS-authenticated peer with a durable trusted-device fingerprint bound by the application trust layer;
- explicit receiver application authorization before any payload bytes are sent;
- source-chunk verification before transmission and receiver-chunk verification before staging;
- transfer-ID, byte-size, and whole-payload-hash binding across sender completion and receiver receipt;
- application composition that can revalidate current account/device/key trust before transfer start, before sender source reads, before receiver staging writes, and before verified return; and
- regression coverage for secure-peer success, pinned-key/device mismatch, handshake identity mismatch, content-digest consistency, manifest structure, binary frame bounds, authenticated payload round trips, receiver rejection, trust binding, mid-transfer trust revocation, chunk corruption, truncation, and trailing-data rejection.

This establishes a bounded Development-stage one-to-one payload movement primitive. It does **not** yet provide durable resume persistence, LAN discovery, production connection/session orchestration, final filesystem placement, complete folder synchronization, or a user-facing Nearby product workflow.

## Current one-to-one payload flow

1. An explicitly trusted device identity is resolved for the peer.
2. A direct peer session is established over the authenticated TLS 1.3 primitive where that transport is selected.
3. The application trust composition layer binds the durable trusted-device fingerprint to the admitted peer and may revalidate current trust at explicit checkpoints.
4. The sender constructs or receives an already validated manifest and creates a fresh random transfer ID.
5. The sender transmits a versioned file/text `PayloadOffer` containing the transfer ID and manifest.
6. Receiver application logic independently authorizes or rejects that exact offered operation. A rejection is returned before payload bytes move and does not expose local policy detail.
7. After acceptance, the sender reads each declared source chunk, verifies it against the manifest, and writes one bounded binary frame.
8. The receiver bounds the declared frame size before allocation, verifies the chunk against the manifest, and only then writes it to the caller-provided staging destination.
9. The sender verifies that the source contains no undeclared trailing data, verifies the whole source digest, and sends a `PayloadCompletion` bound to transfer ID, size, and hash.
10. The receiver validates the completion record and its independently accumulated whole-payload digest.
11. Only after successful verification does the receiver send a `PayloadReceipt` with status `verified`, bound to the same transfer ID, size, and hash.
12. The sender accepts success only after validating that exact verified receipt.

Protocol, framing, stream, staging-write, and integrity failures close the local peer fail-closed so a potentially desynchronized stream is not silently reused.

Transport authentication is not application authorization. The expected trusted peer identity must originate from durable account-scoped trust state or an equivalent approved authority before the secure peer primitive is used, and the receiver application must separately approve the offered payload operation.

## Transfer control records

### PayloadOffer

The current Development offer contains:

- payload protocol version;
- 128-bit random transfer ID encoded as 32 lowercase hexadecimal characters;
- payload kind: `file` or `text`; and
- the complete validated transfer manifest.

The control record itself is subject to the existing 1 MiB JSON-frame maximum and strict unknown-field/trailing-value rejection.

### PayloadDecision

The receiver returns the same protocol version and transfer ID plus an accepted boolean. A rejection intentionally contains no free-form policy reason so application authorization details are not disclosed to the remote peer.

### PayloadCompletion

After all declared chunks are written and the sender independently verifies its source boundary and whole-payload digest, it sends the exact transfer ID, declared size, and manifest hash. A completion mismatch fails closed.

### PayloadReceipt

The receiver emits `verified` only after every declared chunk and the complete payload hash have been validated. The receipt repeats the exact transfer ID, size, and hash. Receipt validation is required before the sender reports success.

A verified receipt proves integrity of the bytes transferred for that manifest. It does not establish backup status, archival retention, destination-path authorization, conflict resolution, or recoverability.

## Current manifest integrity contract

The Development manifest records a version, filename/logical payload name, exact byte size, configured chunk size, whole-payload SHA-256 digest, and ordered chunk entries. Each chunk entry records its zero-based index, exact size, and SHA-256 digest.

The current implementation:

- requires canonical lowercase SHA-256 digest text;
- requires contiguous chunk indexes beginning at zero;
- requires every non-final chunk to use the configured chunk size;
- limits the configured chunk size to the current 1 MiB `DefaultChunkSize` bound;
- requires chunk byte totals to exactly match the declared payload size;
- treats an empty payload as zero chunks with the SHA-256 digest of an empty payload;
- verifies every source chunk before transmission;
- verifies every received chunk before staging;
- rejects undeclared source bytes beyond the manifest payload boundary; and
- independently verifies the complete payload digest on both sender and receiver paths.

This remains a source contract, not a stabilized network compatibility promise.

## Framing and resource bounds

JSON control messages and binary payload chunks use a four-byte big-endian length prefix. The global frame limit is 1 MiB.

Binary receive additionally receives the expected manifest chunk size as a semantic maximum. A peer that declares a frame larger than that expected chunk or larger than the global limit is rejected before payload allocation.

Frame writers continue until all bytes are written and fail if the underlying writer makes zero progress, preventing a successful return after a silent short write.

These are local framing bounds only. Production connection concurrency, total transfer quotas, memory/disk quotas, bandwidth limiting, abuse controls, and global resource admission remain pending.

## Application authorization and staging boundary

`PeerConn.ReceiveTransferPayload` requires a caller-supplied authorization callback. No payload byte is transmitted by the compliant sender until the receiver has returned an explicit acceptance decision.

The receiver takes an `io.Writer` staging destination rather than selecting a filesystem path itself. The transport may have written partial verified chunks when a later stream or integrity failure occurs. Therefore the caller must treat the destination as uncommitted staging and publish/rename/commit it only after the function returns a verified receipt.

This separation intentionally keeps destination-path authorization, path normalization, symlink escape resistance, overwrite/conflict policy, atomic publication, and recovery policy outside the primitive until those controls are implemented explicitly.

## Current-trust revalidation boundary

A payload stream requires the durable trusted-device fingerprint to be bound to the authenticated peer. `SecurePeerFactory` can additionally compose the transfer with the current account-scoped trust store.

The current application composition revalidates trust:

- before transfer start;
- before sender source reads, which normally provides a checkpoint at each chunk boundary;
- before receiver staging writes; and
- before returning a verified operation result.

Revocation or key replacement observed at one of those checkpoints closes the local peer and stops later work. These checks are synchronous and checkpoint-based. They do not continuously monitor the trust store and cannot retroactively stop bytes already written after a successful checkpoint.

## Required properties before production/Stable transfer acceptance

- production trusted-device lookup and secure connection-admission orchestration;
- explicit pairing, approval, revocation, recovery, and reconnect behavior;
- durable resumable transfer state and interrupted-transfer recovery;
- production replay and freshness semantics for transfer operations;
- bounded resource use, rate limits, quotas, and connection concurrency;
- explicit destination/path authorization, path confinement, symlink escape resistance, overwrite policy, and atomic publication;
- transfer cancellation, prioritization, progress, and history where required;
- direct/LAN operation without requiring a hosted GoreeCloud relay;
- privacy-conscious metadata handling;
- complete folder-transfer and folder-synchronization authorization;
- migration/rollback, deployment, and exact-release validation.

## Security boundary

The direct-peer encrypted transport uses Go's reviewed TLS 1.3 implementation rather than a GoreeCloud-invented cryptographic construction. The current wrapper pins the expected Ed25519 device key and identity and requires mutual certificate possession.

The payload protocol uses SHA-256 integrity metadata and random transfer identifiers, but those mechanisms are not presented as a complete application-layer anti-replay/freshness system. Broader replay and stale-operation resistance remains a separate production requirement.

The secure peer and one-to-one payload primitives do not define future Share E2EE, relay-independent confidentiality, complete transport/session replay policy, folder-transfer authorization, or final production admission semantics. Those remain subject to the repository threat model and dedicated security validation.

Network reachability never establishes GoreeCloud Sync authorization, a successful TLS session never grants application data permission by itself, successful payload delivery never establishes backup/recovery authority, and a verified receipt does not authorize a final storage destination.
