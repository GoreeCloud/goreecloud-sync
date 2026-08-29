# GoreeCloud Sync Protocol

## Status

**Pre-stabilization.** Source-level protocol and replication primitives exist, but no GoreeCloud Sync wire protocol is approved as a Stable compatibility promise or production security boundary.

`GC-SYNC/1` is a development protocol identifier used by the current bounded control-frame and peer-transport foundation. It must not be treated as a frozen public protocol version.

## Implemented source foundations

Current source includes:

- bounded length-prefixed JSON control frames with a 1 MiB maximum, strict field decoding, truncated-frame rejection, and trailing-value rejection;
- capability-handshake validation for protocol identifiers, device IDs, feature counts, feature names, and duplicate capabilities;
- TCP peer-stream helpers for exchanging the pre-stabilization capability handshake;
- Ed25519 device identity, public-key fingerprinting, and pairing proofs bound to a device ID and one-time challenge;
- first-party dataset capability negotiation with independent read, write, and delete permissions and highest mutually compatible schema selection;
- versioned record envelopes carrying dataset, schema version, record ID, revision, timestamp, origin device, tombstone state, and application-owned payload;
- deterministic record conflict resolution;
- record-bound proof and authenticated peer/session foundations;
- Privacy Shield purpose/consent and Wardveil trust decision boundaries before durable record acceptance;
- payload-free tombstones, observation receipts, convergence/compaction foundations, and persistent replay high-water state;
- authenticated ingestion and retrieval handlers for current Browser, Search, and Bookmarks datasets;
- deterministic retrieval pagination ordered by bounded record ID using `limit` and exclusive `after` continuation.

These primitives prove source progress only. They do not prove end-to-end production transport security, complete multi-device convergence, deployment acceptance, or Stable compatibility.

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
- treat key possession alone as permission to read or write application data;
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
Authenticated encrypted session (production design pending)
        |
LAN | private network | direct Internet | relay
```

Application ownership remains explicit. GoreeCloud Sync coordinates transport, authorization evidence, convergence, and replication metadata; it does not take ownership of Browser, Search, or Bookmarks payload semantics.

## Identity and pairing

Current source can generate Ed25519 device identity material, derive stable public-key fingerprints, and verify a signed pairing proof bound to device ID and challenge.

That proves possession of the corresponding private key only. Explicit trust approval, challenge expiry/replay enforcement across the final pairing flow, persistent trusted-device state, revocation UX, and final authenticated-encryption session establishment remain separate requirements.

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

Record IDs are currently limited to 512 bytes because they participate in persistence, signatures, deterministic ordering, and exclusive retrieval cursors.

Tombstones intentionally contain no application payload. Delete permission is negotiated separately from read/write permission.

Current authenticated retrieval is ordered by `recordId` and uses bounded pages. The server default page size is 256 records and the accepted maximum is 1,024. `after` is an exclusive record-ID cursor; `nextAfter` is returned only when another page exists.

## Transfer manifests and file synchronization

The transfer-engine foundation contains integrity and framing primitives, but a complete file/folder synchronization protocol is not yet implemented.

A future stable transfer manifest must describe the minimum information required to verify and resume an operation, including version, transfer identifier, payload type, size information, chunking information, integrity identifiers, and authorized operation. Metadata exposure must be minimized.

## Integrity

Chunk verification and final payload verification are separate states. A completed network transfer cannot be reported as verified until the applicable integrity checks succeed.

SHA-256 is currently used for implemented content-digest and fingerprint foundations where recorded by source. Final authenticated-encryption, key-agreement, and key-derivation choices remain security-review decisions and must use reviewed standard cryptographic dependencies rather than GoreeCloud-invented primitives.

## Compatibility

Protocol changes must use explicit version/capability negotiation. Breaking changes must not be shipped under an unchanged compatibility identifier once a Stable protocol exists.

Until stabilization, current development identifiers, schemas, and route contracts may change through reviewed migrations. Source consumers must fail closed on incompatible dataset/schema or continuation behavior.

## Security review gate

No temporary-share E2EE claim, production peer-security claim, Syncthing-replacement claim, or Stable sync-protocol claim may be made until the concrete protocol specification, threat model, cryptographic dependency choices, test vectors, downgrade behavior, replay/expiry protections, trust approval and revocation semantics, durable authorization model, migration/rollback path, and target-environment evidence have been reviewed and validated.
