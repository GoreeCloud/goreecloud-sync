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
  +-- secure-peer trust composition
  |     +-- durable fingerprint binding
  |     +-- explicit current-trust revalidation
  |     +-- operation and bounded-sequence guards
  |
  +-- transfer engine
  |     +-- versioned manifests
  |     +-- per-chunk / whole-payload SHA-256 verification
  |     +-- file/text payload control records
  |     +-- random transfer IDs
  |
  +-- peer transport
        +-- raw TCP DialPeer / AcceptPeer (no trust implied)
        +-- TLS 1.3 DialSecurePeer / AcceptSecurePeer
        |     +-- mutual Ed25519 key possession
        |     +-- exact trusted device-ID/public-key pinning
        |     +-- GoreeCloud Sync ALPN
        |     +-- GC-SYNC handshake identity binding
        |
        +-- authenticated one-to-one payload stream
              +-- explicit receiver application authorization
              +-- bounded binary chunk frames
              +-- source and receiver integrity checks
              +-- staging-only receiver output
              +-- verified completion receipt
```

The default `serve` command exposes the base development HTTP service and does not automatically compose every replication, trust, secure-peer, or payload-transfer component above.

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

The current payload path adds another independent application decision: the receiver must authorize the exact offered file/text transfer before a compliant sender moves payload bytes. Device trust is therefore necessary but not sufficient transfer authorization.

A network route, LAN discovery event, NetBird membership, successful raw TCP connection, valid TLS session, durable device trust, or successful capability handshake must never grant application data or destination access by itself.

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

The private key remains owned by the higher-level protected key boundary; transport does not persist it. The expected remote identity must come from explicit trusted-device authorization.

### Authenticated one-to-one payload stream

`PeerConn.SendTransferPayload` and `PeerConn.ReceiveTransferPayload` provide a bounded Development-stage data stream only when the peer has both a TLS-authenticated remote device identity and a durable trusted-device fingerprint bound by the application trust layer.

The stream uses:

- a versioned offer containing payload kind, random transfer ID, and validated manifest;
- a receiver decision before any payload bytes move;
- binary chunk frames bounded by both the 1 MiB global ceiling and the exact expected manifest chunk size before allocation;
- source-chunk verification before send;
- receiver-chunk verification before staging;
- sender completion bound to transfer ID, size, and whole-payload hash; and
- a receiver receipt marked verified only after independent whole-payload verification.

Application composition can revalidate current durable trust before transfer start, immediately before receiver authorization, before sender source reads, before receiver staging writes, and before returning verified local success. These are explicit checkpoints, not continuous asynchronous revocation.

The receiver accepts a staging `io.Writer`; it does not choose or authorize a final filesystem path. Final path policy, confinement, overwrite/conflict handling, and atomic publication remain separate future components.

Production lookup, listener admission, discovery, redial/revocation behavior, complete connection lifecycle, durable resume state, and broader replay/freshness policy remain incomplete.

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
  +-- Dataset/folder/transfer policy
  +-- Transfer coordination and resume state
  +-- Share lifecycle
  +-- Conflict management
  +-- Audit/event boundary
  |
Transfer engine
  |
  +-- discovery
  +-- secure connection selection
  +-- chunking and durable resume
  +-- integrity validation
  +-- destination/path safety
  +-- progress / cancellation / rate controls
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

Synchronization and one-to-one transfer provide movement and availability. Everkeep, GoreeCloud Backup, filesystem snapshots, and other independent recovery systems provide recovery authority.

A verified payload receipt means the staged bytes matched the offered manifest for that transfer. It does not make those bytes a backup, establish retention, or prove a recoverable independent copy.

Live databases, active application-managed volumes, certificate stores, backup repositories, and other consistency-sensitive state require application-aware protection rather than ordinary file synchronization.

## Technology direction

- Go: service, CLI, concurrency-heavy transfer engine, APIs, TLS peer transport.
- TypeScript: future Glaze UI web administration client.
- Kotlin: future Android client and background transfer services.
- Rust: selective security-sensitive or performance-critical components when justified.
- SQL: relational state when durable multi-user metadata or resume state requires it.

Technology choices remain revisable when implementation evidence shows a safer or simpler option.

## Current claim boundary

The architecture contains source foundations for device trust, current-trust revalidation, first-party record replication, raw peer TCP, TLS 1.3 authenticated peer transport, transfer manifests/integrity verification, and an authenticated one-to-one file/text payload stream with explicit receiver authorization and staging-only output. Production account/runtime composition, LAN discovery, durable resume, final filesystem publication, complete folder synchronization, user-facing administration, multi-user authorization, Nearby, Share, deployment, and Stable acceptance remain pending.
