# GoreeCloud Sync Transfer Protocol

## Status

Development foundation only. `GC-SYNC/1` is an internal pre-stabilization identifier and is not an approved stable wire protocol or production compatibility promise.

## Purpose

This document records the first transfer-engine primitives being developed for GoreeCloud Sync Milestone 1.

## Current source foundation

The current branch provides:

- device Ed25519 key-generation primitives;
- public-key fingerprinting;
- chunk and transfer-session data models;
- SHA-256 content-digest helpers;
- an initial transport handshake model;
- regression coverage ensuring byte-slice and streaming digest helpers agree.

It does not yet provide a network transfer implementation.

## Planned transfer flow

1. An enrolled device presents its device identity.
2. Application authorization independently approves the requested transfer operation.
3. A transfer session and manifest are established.
4. Payload data is split into bounded chunks.
5. Chunks are transferred through an authenticated encrypted session.
6. Received chunks are integrity-checked and resumable progress is recorded.
7. Final payload integrity is verified before completion is reported.

## Required properties before Milestone 1 acceptance

- authenticated encrypted one-to-one transfer;
- explicit pairing and revocation behavior;
- resumable transfer state;
- chunk and final-payload integrity verification;
- no silent corruption or silent downgrade;
- replay and tampering resistance appropriate to the selected mature cryptographic primitives;
- bounded resource use;
- direct/LAN operation without requiring a hosted GoreeCloud relay;
- privacy-conscious metadata handling.

## Security boundary

This document does not define a new cryptographic construction. Concrete key agreement, authenticated encryption, key derivation, replay protection, protocol transcript binding, and downgrade behavior must use mature reviewed primitives and remain subject to the repository threat model and dedicated security validation.

Network reachability never establishes GoreeCloud Sync authorization, and successful byte delivery never establishes backup or recovery authority.
