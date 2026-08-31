# Architecture

## Status

Active Development — pre-Stable. This document describes current implemented source boundaries plus planned product architecture. It does not claim complete production synchronization, Nearby, Share, or deployment acceptance.

## Product model

GoreeCloud Sync combines three separate workflows behind one authorization and transfer platform:

1. **Sync** — durable replication of approved application datasets and, later, approved folders.
2. **Nearby** — immediate device-to-device transfer.
3. **Share** — temporary encrypted delivery.

These modes share device identity, user authorization, integrity verification, transport selection, observability, and policy infrastructure but must remain independently controllable.

## Current source architecture

```text
Development service / application composition
  |
  +-- authenticated HTTP peer/session resolver
  |     +-- account-scoped trusted-device enforcement
  |
  +-- first-party dataset ingestion/retrieval
  |     +-- Browser
  |     +-- Search
  |     +-- Bookmarks
  |
  +-- device identity and pairing
  |     +-- protected Ed25519 device keys
  |     +-- signed pairing proofs
  |     +-- one-time pairing challenges
  |     +-- durable trusted-device authorization / revocation
  |
  +-- peer transport
        +-- raw TCP DialPeer / AcceptPeer (no trust implied)
        +-- TLS 1.3 DialSecurePeer / AcceptSecurePeer
              +-- mutual Ed25519 key possession
              +-- exact trusted device-ID/public-key pinning
              +-- GoreeCloud Sync ALPN
              +-- GC-SYNC handshake identity binding
```

The default `serve` command exposes the base development HTTP service and does not automatically compose every replication, trust, or secure-peer component above.

## Authorization model

The intended authorization hierarchy remains:

```text
Person -> Account -> Device -> Explicit capability grants
                         |
                         +-- dataset grants
                         +-- folder grants
                         +-- transfer grants
                         +-- share grants
```

Current source has account-scoped durable trusted-device state plus an HTTP peer resolver that checks exact device ID and key fingerprint. Direct secure-peer transport takes an already-trusted expected device ID and raw Ed25519 public key from higher-level code and proves that key/identity over TLS 1.3.

A network route, LAN discovery event, NetBird membership, successful raw TCP connection, valid TLS session, or successful capability handshake must never grant application data access by itself.

Planned folder permissions include read, write, contribute, receive-only, drop-only, and administrative management. Exact permission semantics must be versioned before protocol stabilization.

## Peer transport layers

### Raw transport

`DialPeer` and `AcceptPeer` are context-bound raw TCP primitives. They intentionally do not assign peer trust or encryption. Keeping this lower-level primitive explicit avoids falsely treating reachability as authentication.

### Authenticated encrypted transport

`DialSecurePeer` and `AcceptSecurePeer` layer TLS 1.3 over TCP using Go's standard cryptographic libraries. The current source contract:

- requires a local canonical device ID and in-memory Ed25519 private key;
- requires the expected remote canonical device ID and exact raw Ed25519 public key;
- presents short-lived self-signed Ed25519 certificates as TLS key-possession carriers;
- requires mutual certificate possession;
- verifies the exact expected remote device ID and public key rather than using Web PKI or hostnames as device trust;
- requires ALPN `goreecloud-sync/1`;
- bounds TLS handshake duration;
- binds subsequent GC-SYNC capability-handshake device IDs to TLS-authenticated identities.

The private key remains owned by the higher-level protected key boundary; transport does not persist it. The expected remote identity must come from explicit trusted-device authorization. Production lookup, listener admission, redial/revocation behavior, connection lifecycle, and broader replay/freshness policy remain incomplete.

## First-party record replication

Current development handlers support negotiated records for:

- `search.history`
- `bookmarks.items`
- `browser.tabs`
- `browser.history`

Application record meaning remains owned by the originating application. Sync coordinates authenticated, authorized movement and deterministic convergence without becoming the semantic owner of Browser, Search, or Bookmarks data.

Retrieval uses bounded record-ID cursor pagination and validates the complete persisted store before a page is emitted, while retaining only a page-bounded candidate window.

## Future logical components

```text
Clients
  |
  +-- CLI
  +-- Glaze UI web administration
  +-- Android
  +-- future platform clients
  |
Application API
  |
  +-- GoreeCloud Identity account integration
  +-- Device trust and secure-session admission
  +-- Dataset/folder policy
  +-- Transfer coordination
  +-- Share lifecycle
  +-- Conflict management
  +-- Audit/event boundary
  |
Transfer engine
  |
  +-- discovery
  +-- secure connection selection
  +-- chunking and resume
  +-- integrity validation
  +-- direct transport
  +-- relay fallback
  |
Storage adapters
```

## Transport preference

For approved endpoints, the expected preference is:

1. direct local connection;
2. direct approved private-network connection;
3. safe direct Internet peer connection when explicitly supported;
4. self-hosted rendezvous or relay fallback.

Raw synchronization ports must not be exposed publicly merely for convenience. Any production peer transport must combine network controls with current device trust and application authorization.

## Data and recovery boundary

Synchronization provides movement and availability. Everkeep, GoreeCloud Backup, filesystem snapshots, and other independent recovery systems provide recovery authority.

Live databases, active application-managed volumes, certificate stores, backup repositories, and other consistency-sensitive state require application-aware protection rather than ordinary file synchronization.

## Technology direction

- Go: service, CLI, concurrency-heavy transfer engine, APIs, TLS peer transport.
- TypeScript: future Glaze UI web administration client.
- Kotlin: future Android client and background transfer services.
- Rust: selective security-sensitive or performance-critical components when justified.
- SQL: relational state when durable multi-user metadata requires it.

Technology choices remain revisable when implementation evidence shows a safer or simpler option.

## Current claim boundary

The architecture contains tested source foundations for device trust, first-party record replication, raw peer TCP, and an authenticated TLS 1.3 peer primitive. Production account/runtime composition, LAN discovery, resumable folder transfer, user-facing administration, multi-user authorization, Nearby, Share, deployment, and Stable acceptance remain pending.
