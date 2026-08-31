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
- raw context-bound TCP peer helpers that do not imply trust;
- a TLS 1.3 authenticated direct-peer primitive for already-trusted device identities;
- mutual Ed25519 key-possession proof with exact expected device-ID/raw-key pinning;
- GoreeCloud Sync ALPN and GC-SYNC capability-handshake identity binding;
- bounded TLS handshake time and session-ticket disabling in the current secure wrapper;
- regression coverage for secure-peer success, pinned-key/device mismatch, handshake identity mismatch, and content-digest consistency.

This does **not** yet provide a complete resumable file/folder transfer implementation or production connection-admission runtime.

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
