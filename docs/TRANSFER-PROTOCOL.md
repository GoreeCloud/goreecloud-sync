# GoreeCloud Sync Transfer Protocol

## Status

Development foundation only. `GC-SYNC/1` is an internal pre-stabilization identifier and is not an approved Stable wire protocol or production compatibility promise.

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
- regression coverage for secure-peer success, pinned-key/device mismatch, handshake identity mismatch, content-digest consistency, manifest structure, chunk corruption, truncation, and trailing-data rejection.

The manifest/verification checkpoint establishes source-level integrity acceptance primitives. It does **not** yet transmit file or text payloads through the secure peer, persist resumable progress, authorize folder paths, or provide a complete file/folder transfer implementation or production connection-admission runtime.

## Intended transfer flow

1. An explicitly trusted device identity is resolved for the peer.
2. A direct peer session is established over the authenticated TLS 1.3 primitive where that transport is selected.
3. Application authorization independently approves the requested transfer operation.
4. A transfer session and manifest are established.
5. Payload data is split into bounded chunks.
6. Chunks are transferred through the authenticated encrypted session.
7. Received chunks are integrity-checked and resumable progress is recorded.
8. Final payload integrity is verified before completion is reported.

Transport authentication is not application authorization. The expected trusted peer identity must originate from durable account-scoped trust state or an equivalent approved authority before the secure peer primitive is used.

## Current manifest integrity contract

The Development manifest records a version, filename, exact byte size, configured chunk size, whole-payload SHA-256 digest, and ordered chunk entries. Each chunk entry records its zero-based index, exact size, and SHA-256 digest.

The current implementation:

- requires canonical lowercase SHA-256 digest text;
- requires contiguous chunk indexes beginning at zero;
- requires every non-final chunk to use the configured chunk size;
- limits the configured chunk size to the current 1 MiB `DefaultChunkSize` bound;
- requires chunk byte totals to exactly match the declared payload size;
- treats an empty file as zero chunks with the SHA-256 digest of an empty payload;
- verifies every received chunk before including it in final-payload verification; and
- rejects undeclared bytes after the exact manifest payload boundary.

This is intentionally a source contract, not yet a stabilized network serialization promise. Durable resume state, transfer-operation freshness/replay semantics, rate controls, filesystem destination safety, and authenticated transport wiring remain separate pending work.

## Required properties before production/Stable transfer acceptance

- production trusted-device lookup and secure connection-admission orchestration;
- explicit pairing, approval, revocation, recovery, and reconnect behavior;
- resumable transfer state;
- chunk and final-payload integrity verification;
- no silent corruption, insecure fallback, or silent downgrade;
- replay and stale-operation resistance appropriate to each transfer operation;
- bounded resource use, rate limits, quotas, and connection concurrency;
- direct/LAN operation without requiring a hosted GoreeCloud relay;
- privacy-conscious metadata handling;
- path confinement and symlink/filesystem escape resistance for folder transfers;
- migration/rollback, deployment, and exact-release validation.

## Security boundary

The direct-peer encrypted transport uses Go's reviewed TLS 1.3 implementation rather than a GoreeCloud-invented cryptographic construction. The current wrapper pins the expected Ed25519 device key and identity and requires mutual certificate possession.

The secure peer primitive does not define future Share E2EE, relay-independent confidentiality, complete transport/session replay policy, folder-transfer authorization, or final production admission semantics. Those remain subject to the repository threat model and dedicated security validation.

Network reachability never establishes GoreeCloud Sync authorization, a successful TLS session never grants application data permission by itself, and successful byte delivery never establishes integrity, backup, or recovery authority.
