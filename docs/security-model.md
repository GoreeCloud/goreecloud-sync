# Security and Privacy Model

## Status

Active Development — pre-Stable. This record distinguishes implemented source controls from incomplete production security controls. Passing source tests or the presence of security primitives does not establish production security acceptance.

## Trust boundaries

GoreeCloud Sync treats users, accounts, devices, network paths, peer transports, application APIs, synchronized data, transfer operations, staging destinations, relay infrastructure, temporary storage, and recovery systems as separate trust boundaries.

Key principles:

- authenticate every device independently;
- authorize every data relationship and transfer operation explicitly;
- keep network reachability and transport authentication separate from application authorization;
- use established cryptographic libraries and protocols rather than custom encryption;
- make revocation a first-class lifecycle operation;
- minimize stored metadata and logs;
- protect device private keys outside ordinary repository/application storage;
- fail closed when identity, trust, negotiated schema, persisted state, secure-session validation, transfer framing, or integrity validation is invalid;
- keep production secrets outside source control.

## Implemented source controls

### Device key and pairing controls

- Ed25519 device identity and SHA-256 public-key fingerprints.
- Device private-key persistence through a `KeyProtector` abstraction; Sync does not write plaintext private-key bytes to disk.
- Signed pairing proofs that bind a canonical device ID and one-time challenge to an Ed25519 public key.
- Cryptographically random, short-lived pairing challenges with exact single consumption, expiry rejection, and replay rejection.
- `VerifiedPairing` values that can only be produced after proof verification and challenge consumption.

### Durable device authorization and current-trust checks

- Account-scoped trusted-device records.
- Explicit authorization from verified pairing state.
- Explicit revocation.
- Refusal to silently replace an active trusted device with different key material.
- Validation that stored raw public key and fingerprint remain consistent.
- Private filesystem permissions and atomic publication for the development trust store.
- A trusted HTTP peer resolver that requires an authenticated peer's exact device ID and key fingerprint to remain trusted for the configured account.
- Secure-peer composition that binds the durable trusted public-key fingerprint to the admitted `PeerConn`.
- Explicit `RevalidatePeer`, one-operation, and bounded operation-sequence checkpoints.
- Payload-transfer composition that revalidates before transfer start, immediately before receiver application authorization, before sender source reads, before receiver staging writes, and before returning verified local success.

These checks are synchronous checkpoints. They do not provide instantaneous asynchronous revocation and cannot retroactively stop I/O that already occurred after a successful checkpoint.

### Authenticated encrypted direct-peer transport

A separate secure-peer transport exists in source for **already-trusted** peers:

- TLS 1.3 is required; earlier TLS versions are not accepted by the secure wrapper.
- Go's reviewed `crypto/tls` and `crypto/x509` implementations perform encrypted transport and TLS CertificateVerify key-possession proof.
- Both sides present short-lived self-signed certificates using their existing Ed25519 device key.
- The certificate is not itself a trust grant. Higher-level code supplies the expected trusted remote device ID and exact raw Ed25519 public key.
- The connection verifies the expected device ID and raw public key, certificate role/validity, self-signature structure, and GoreeCloud Sync ALPN.
- TLS handshake duration is bounded.
- Session tickets are disabled in the current secure wrapper.
- After TLS authentication, GC-SYNC capability-handshake device IDs must match the identities bound to the secure connection.
- Transport never persists the private key it receives in memory.

Raw `DialPeer` and `AcceptPeer` remain intentionally unauthenticated lower-level TCP primitives. They must not be treated as secure sessions.

### Authenticated one-to-one payload controls

The current Development payload primitive supports one file or logical text payload over an already authenticated and durable-trust-bound peer:

- a versioned `PayloadOffer` carries a fresh cryptographically random 128-bit transfer ID and validated manifest;
- a receiver application-authorizer must explicitly accept the offered operation before the compliant sender transmits payload bytes;
- policy rejection records expose only accepted/rejected state, not local authorization detail;
- payload chunks use bounded length-prefixed binary frames with a 1 MiB global ceiling and an exact expected-manifest-chunk allocation ceiling;
- the sender verifies every source chunk against the manifest before transmission;
- the receiver verifies every received chunk before writing it to caller-provided staging;
- the sender rejects source truncation, source corruption, and undeclared trailing data;
- sender completion is bound to transfer ID, byte size, and whole-payload hash;
- the receiver independently verifies the complete payload before sending a `verified` receipt;
- the sender accepts success only when the receipt matches the exact transfer ID, size, and hash; and
- protocol, stream, framing, staging-write, or integrity failures close the local peer fail-closed.

The staging writer is not a final storage authority. Callers must not publish partially received bytes or treat a verified receipt as final path authorization, backup evidence, or recovery evidence.

The random transfer ID and ordered stream binding are not a complete application-layer anti-replay/freshness protocol. Durable transfer replay state, resume sequencing, and production freshness semantics remain pending.

### Application-record acceptance controls

- Exact negotiated dataset/schema checks for first-party application records.
- Payload-required live records and payload-free deletion tombstones.
- Record-bound proof foundations and replay/high-water controls.
- Privacy Shield and Wardveil-facing acceptance decision boundaries before durable record acceptance.
- Full persisted-store validation before paginated retrieval so corruption outside a requested page remains fail-closed.

## Security properties not yet complete

The current secure-peer and payload primitives are not the same as complete production secure-session and transfer architecture. Still required are:

- production account/trusted-device lookup and connection-admission orchestration;
- secure listener/dial selection, address discovery, and lifecycle management;
- approved revocation-aware reconnect and full revalidation cadence;
- broader transport/session/transfer replay and freshness semantics beyond TLS, random transfer IDs, and one-time pairing-challenge replay protection;
- durable transfer resume state and interrupted-transfer recovery;
- final destination/path authorization, canonicalization, root confinement, symlink safety, overwrite/conflict policy, and atomic publication;
- production multi-user account and folder authorization;
- transfer cancellation, progress, rate limiting, quotas, connection concurrency controls, abuse handling, and operational security telemetry;
- user-facing pairing approval, trust inspection, revocation, recovery, key-replacement, and Nearby transfer UX;
- complete GoreeCloud Identity, Wardveil Security, Privacy Shield, Everkeep, GoreeCloud Mesh, and Glaze UI production integration evidence;
- deployment and release acceptance.

## Threat classes

The implementation and production design must account for at least:

- unauthorized or stolen devices;
- compromised accounts;
- malicious or untrusted peers;
- pairing, session, record, deletion, or transfer replay;
- protocol downgrade attempts;
- TLS peer-key or identity substitution;
- tampering with transferred chunks or source content changing after manifest creation;
- path traversal and unsafe final file publication;
- relay or temporary-storage compromise;
- accidental public service exposure;
- abusive share-link enumeration or download attempts;
- conflict-driven data loss;
- destructive deletion propagation;
- resource exhaustion;
- sensitive information leakage through logs or diagnostics.

## Peer transport and transfer security boundary

A valid TLS session proves possession of the pinned device key and encrypts/authenticates the transport. It does **not** by itself grant account, dataset, folder, transfer, destination, or share authorization.

The expected remote identity must be resolved from explicit trust state by higher-level code. A self-signed certificate from an unknown key, a matching device name without the pinned key, LAN presence, NetBird membership, or raw TCP reachability is insufficient.

The current secure wrapper intentionally uses exact key pinning instead of public Web PKI hostname trust because GoreeCloud device authorization is account/device based. The client disables default Web-PKI verification only while replacing it with strict `VerifyConnection` checks for the exact authorized device identity and key.

For the one-to-one payload path, application policy independently approves or rejects the exact offer. The receiver writes only to staging. No transport record authorizes a final path or overwrite.

## Share security

Temporary sharing is intended to use client-side end-to-end encryption. A storage or relay service must not need plaintext file contents to perform its role. Exact key agreement, encryption, metadata-protection, recipient authentication, expiry, and revocation behavior remain subject to separate protocol security review and implementation.

The direct-peer TLS and one-to-one payload primitives do not by themselves satisfy Share's end-to-end encryption requirements for relay-backed delivery.

## Local defaults

Development services bind to loopback by default. Public exposure must be an explicit deployment decision protected by appropriate GoreeCloud publication, firewall, authentication, reverse-proxy, rate-limit, and application-authorization controls.

The default `serve` command does not enable production secure-peer listeners, discovery, payload-transfer orchestration, or final storage publication.

## Logging

Logs must not include file/text contents, encryption keys, authentication secrets, private device keys, reusable bearer credentials, or unnecessary full paths. Operational identifiers should be scoped and minimized.

Secure-peer and payload-transfer failures should identify the class of failure needed for operations without logging certificate private material, raw secrets, payload contents, or unnecessary local authorization details.

## Recovery safety

Security controls must not undermine recoverability. Revocation, key rotation, conflict handling, deletion propagation, failed partial staging, final file publication, and future encrypted-session/account recovery must have documented recovery behavior before production approval.

Everkeep remains the continuity and recovery authority; synchronized copies or verified transfers do not become backups merely because transport is authenticated, encrypted, and integrity-checked.

## Stable security gate

Before Stable, evidence must cover at minimum:

- multi-user isolation and application/folder/transfer authorization;
- device enrollment, revocation, recovery, and key replacement;
- secure-session connection admission and revocation behavior;
- TLS downgrade/key-substitution/identity-binding tests;
- replay and stale-operation behavior across transfer/session scopes;
- transfer integrity, staging behavior, final destination safety, and interrupted-transfer recovery;
- conflict and deletion safety;
- path traversal and filesystem escape resistance for file/folder publication;
- resource exhaustion, transfer quotas/rate limits, timeout/cancellation policy, and bounded concurrency;
- logging privacy and secret separation;
- Share E2EE/expiry/revocation behavior where claimed;
- dependency/security review;
- migration, rollback, packaging, deployment, and exact-release validation;
- applicable Wardveil Security, Privacy Shield, Everkeep, GoreeCloud Identity, GoreeCloud Mesh, and Glaze UI acceptance evidence.

## Current claim boundary

GoreeCloud Sync now contains source controls for pairing replay/expiry, durable trusted-device authorization, trust-enforced peer resolution, explicit current-trust checkpoints, TLS 1.3 authenticated direct-peer transport, bounded transfer manifests/integrity verification, and an authenticated one-to-one file/text payload stream with independent receiver authorization and verified completion receipts. Complete production transfer replay/freshness, resume, final storage publication, session/runtime orchestration, folder synchronization, multi-user authorization, Nearby, Share, deployment, and Stable security acceptance remain incomplete.
