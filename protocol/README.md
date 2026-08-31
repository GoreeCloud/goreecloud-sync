# GoreeCloud Sync Protocol

## Status

**Pre-stabilization.** Source-level protocol, replication, device-trust, and secure-peer primitives exist, but no GoreeCloud Sync wire protocol is approved as a Stable compatibility promise or complete production security boundary.

`GC-SYNC/1` is a development protocol identifier used by the current bounded control-frame and peer-transport foundation. It must not be treated as a frozen public protocol version.

## Implemented source foundations

Current source includes:

- bounded length-prefixed JSON control frames with a 1 MiB maximum, strict field decoding, truncated-frame rejection, and trailing-value rejection;
- capability-handshake validation for protocol identifiers, device IDs, feature counts, feature names, and duplicate capabilities;
- raw TCP peer-stream helpers for exchanging the pre-stabilization capability handshake without implying trust;
- Ed25519 device identity, public-key fingerprinting, and pairing proofs bound to a device ID and one-time challenge;
- cryptographically random short-lived pairing challenges with exact consumption, expiry rejection, and replay rejection;
- durable account-scoped trusted-device authorization and explicit revocation foundations;
- a TLS 1.3 secure-peer primitive for already-trusted peers using Go `crypto/tls` and `crypto/x509`, mutual Ed25519 key possession, exact trusted device-ID/raw-key pinning, GoreeCloud Sync ALPN, bounded handshake time, a TLS-protected server acceptance confirmation, and GC-SYNC handshake identity binding;
- first-party dataset capability negotiation with independent read, write, and delete permissions and highest mutually compatible schema selection;
- versioned record envelopes carrying dataset, schema version, record ID, revision, timestamp, origin device, tombstone state, and application-owned payload;
- deterministic record conflict resolution;
- record-bound proof and authenticated peer/session foundations;
- Privacy Shield purpose/consent and Wardveil trust decision boundaries before durable record acceptance;
- payload-free tombstones, observation receipts, convergence/compaction foundations, and persistent replay high-water state;
- authenticated ingestion and retrieval handlers for current Browser, Search, and Bookmarks datasets;
- deterministic retrieval pagination ordered by bounded record ID using `limit` and exclusive `after` continuation.

These primitives prove source progress only. They do not prove complete production secure-session orchestration, complete multi-device convergence, deployment acceptance, or Stable compatibility.

## Goals

The protocol layer is intended to support:

- authenticated device identity;
- explicit user, application-dataset, folder, and share authorization;
- persistent synchronization;
- nearby direct transfer;
- temporary encrypted sharing;
- resumable chunk transfer;
- integrity verification;
- capability negotiation;
- transport independence;
- controlled relay fallback;
- forward-compatible version negotiation.

## Non-goals

The protocol must not:

- invent new cryptographic primitives;
- assume network membership equals authorization;
- treat key possession or a successful TLS connection alone as permission to read or write application data;
- allow clients to self-assert Privacy Shield consent or Wardveil trust decisions;
- silently discard file or record conflicts;
- retain deleted application payload merely to communicate deletion;
- treat byte delivery or synchronization as proof of Everkeep recoverability;
- require a permanently hosted GoreeCloud cloud account;
- depend on one transport provider or relay operator.

## Current layered direction

```text
Application operation
  Sync | Nearby | Share
        |
Application-owned data semantics
  Browser | Search | Bookmarks | future datasets
        |
Authenticated authorization + capability negotiation
        |
Record / transfer manifest + integrity contract
        |
Conflict, deletion, replay, resume state
        |
TLS 1.3 authenticated direct-peer primitive
  (already-trusted device identity; production admission/orchestration pending)
        |
LAN | private network | direct Internet | future relay
```

Application ownership remains explicit. GoreeCloud Sync coordinates transport, authorization evidence, convergence, and replication metadata; it does not take ownership of Browser, Search, or Bookmarks payload semantics.

Raw peer TCP remains a separate lower-level primitive and must not be treated as an authenticated encrypted session.

## Identity and pairing

Current source can generate Ed25519 device identity material, derive stable public-key fingerprints, verify a signed pairing proof bound to device ID and challenge, enforce one-time challenge expiry/replay behavior, and authorize the resulting verified pairing into durable account-scoped trusted-device state.

Pairing proof demonstrates possession of the corresponding private key but does not itself authorize application data access. Durable trusted-device state remains a separate explicit approval step, and runtime trusted-peer enforcement remains separate from transport encryption.

The direct secure-peer primitive accepts an already-authorized expected remote device ID and raw Ed25519 public key from higher-level code. It uses TLS 1.3 to prove key possession and protect the direct transport. On the accepting side, the constructor returns only after the client certificate has passed exact device/key validation; it then sends a bounded TLS-protected ready confirmation containing the server identity and the client identity it accepted. The dialing constructor validates that confirmation before returning, so a locally completed TLS 1.3 client handshake is not mistaken for proof that the server accepted the client identity. Subsequent GC-SYNC capability handshakes are rejected if they claim a different device identity.

Production account/trusted-device lookup, connection admission, reconnect/revocation behavior, user-facing approval/recovery/key-replacement UX, and broader session freshness policy remain incomplete.

## First-party record replication

The current record envelope contains:

- dataset identifier;
- negotiated schema version;
- bounded record ID;
- monotonic application revision;
- update timestamp;
- origin device;
- deletion/tombstone state;
- application-owned payload for live records.

Record IDs are currently limited to 512 bytes because they participate in persistence, signatures, deterministic ordering, and exclusive retrieval cursors. First-party clients must enforce the same bound before signing or submitting a record and must reject retrieved `recordId`, `after`, or `nextAfter` values beyond that bound before they become continuation or application state.

First-party producers and consumers must fail closed unless a record's dataset and schema version exactly match the negotiated application capability. Live records must contain application payload; tombstones must contain none, and clients must not sign, submit, or accept a tombstone that retains deleted application data. Delete permission remains negotiated separately from read/write permission.

A first-party submission client that accepts a `RecordProof` must also fail closed before transport unless the proof device ID equals the record `originDevice`, the public key and signature are valid raw-URL Ed25519 encodings with the required key/signature lengths, and the signature verifies over the exact `GC-SYNC-RECORD/1` message for that record. Client-side proof preflight prevents malformed, mismatched, or post-signing-mutated records from reaching transport; it does not replace server verification against the authenticated peer identity, key fingerprint, replay state, Privacy Shield decision, or Wardveil trust decision.

Current authenticated retrieval is ordered by `recordId` and uses bounded pages. The server default page size is 256 records and the accepted maximum is 1,024. `after` is an exclusive record-ID cursor; `nextAfter` is returned only when another page exists.

## Transfer manifests and file synchronization

The transfer-engine foundation contains integrity and framing primitives, but a complete file/folder synchronization protocol is not yet implemented.

A future stable transfer manifest must describe the minimum information required to verify and resume an operation, including version, transfer identifier, payload type, size information, chunking information, integrity identifiers, and authorized operation. Metadata exposure must be minimized.

## Integrity and encryption

Chunk verification and final payload verification are separate states. A completed network transfer cannot be reported as verified until the applicable integrity checks succeed.

SHA-256 is currently used for implemented content-digest and fingerprint foundations where recorded by source.

For already-trusted direct peers, the current secure-peer source primitive uses TLS 1.3 from Go's reviewed standard library for transport confidentiality, integrity, and mutual key-possession proof. This does not by itself define production connection admission, complete replay/freshness policy, future relay security, or Share end-to-end encryption.

Future Share E2EE, relay-independent confidentiality, and any additional key-agreement or key-derivation requirements must use reviewed standard cryptographic dependencies rather than GoreeCloud-invented primitives.

## Compatibility

Protocol changes must use explicit version/capability negotiation. Breaking changes must not be shipped under an unchanged compatibility identifier once a Stable protocol exists.

Until stabilization, current development identifiers, schemas, secure-peer contracts, and route contracts may change through reviewed migrations. Source consumers must fail closed on incompatible dataset/schema, identity, or continuation behavior.

## Security review gate

No temporary-share E2EE claim, complete production peer-security claim, Syncthing-replacement claim, or Stable sync-protocol claim may be made until the concrete protocol specification, threat model, cryptographic dependency choices, test vectors, downgrade behavior, replay/expiry protections, trust approval and revocation semantics, durable authorization model, production connection-admission/runtime composition, migration/rollback path, and target-environment evidence have been reviewed and validated.
