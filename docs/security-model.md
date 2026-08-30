# Security and Privacy Model

## Status

Active Development — pre-Stable. This record distinguishes implemented source controls from incomplete production security controls. Passing source tests or the presence of security primitives does not establish production security acceptance.

## Trust boundaries

GoreeCloud Sync treats users, accounts, devices, network paths, peer transports, application APIs, synchronized data, relay infrastructure, temporary storage, and recovery systems as separate trust boundaries.

Key principles:

- authenticate every device independently;
- authorize every data relationship explicitly;
- keep network reachability separate from application authorization;
- use established cryptographic libraries and protocols rather than custom encryption;
- make revocation a first-class lifecycle operation;
- minimize stored metadata and logs;
- protect device private keys outside ordinary repository/application storage;
- fail closed when identity, trust, negotiated schema, persisted state, or secure-session validation is invalid;
- keep production secrets outside source control.

## Implemented source controls

### Device key and pairing controls

- Ed25519 device identity and SHA-256 public-key fingerprints.
- Device private-key persistence through a `KeyProtector` abstraction; Sync does not write plaintext private-key bytes to disk.
- Signed pairing proofs that bind a canonical device ID and one-time challenge to an Ed25519 public key.
- Cryptographically random, short-lived pairing challenges with exact single consumption, expiry rejection, and replay rejection.
- `VerifiedPairing` values that can only be produced after proof verification and challenge consumption.

### Durable device authorization

- Account-scoped trusted-device records.
- Explicit authorization from verified pairing state.
- Explicit revocation.
- Refusal to silently replace an active trusted device with different key material.
- Validation that stored raw public key and fingerprint remain consistent.
- Private filesystem permissions and atomic publication for the development trust store.
- A trusted HTTP peer resolver that requires an authenticated peer's exact device ID and key fingerprint to remain trusted for the configured account.

### Authenticated encrypted direct-peer transport

A separate secure-peer transport now exists in source for **already-trusted** peers:

- TLS 1.3 is required; earlier TLS versions are not accepted by the secure wrapper.
- Go's reviewed `crypto/tls` and `crypto/x509` implementations perform the encrypted transport and TLS CertificateVerify key-possession proof.
- Both sides present short-lived self-signed certificates using their existing Ed25519 device key.
- The certificate is not itself a trust grant. Higher-level code supplies the expected trusted remote device ID and exact raw Ed25519 public key.
- The connection verifies the expected device ID and raw public key, certificate role/validity, self-signature structure, and GoreeCloud Sync ALPN.
- TLS handshake duration is bounded.
- Session tickets are disabled in the current secure wrapper.
- After TLS authentication, GC-SYNC capability-handshake device IDs must match the identities bound to the secure connection.
- Transport never persists the private key it receives in memory.

Raw `DialPeer` and `AcceptPeer` remain intentionally unauthenticated lower-level TCP primitives. They must not be treated as secure sessions.

### Application-record acceptance controls

- Exact negotiated dataset/schema checks for first-party application records.
- Payload-required live records and payload-free deletion tombstones.
- Record-bound proof foundations and replay/high-water controls.
- Privacy Shield and Wardveil-facing acceptance decision boundaries before durable record acceptance.
- Full persisted-store validation before paginated retrieval so corruption outside a requested page remains fail-closed.

## Security properties not yet complete

The secure-peer primitive is not the same as complete production secure-session architecture. Still required are:

- production account/trusted-device lookup and connection-admission orchestration;
- secure listener/dial selection and lifecycle management;
- revocation-aware reconnect and stale-authorization policy;
- broader transport/session replay and freshness semantics beyond TLS and one-time pairing-challenge replay protection;
- production multi-user account and folder authorization;
- rate limiting, quotas, connection concurrency controls, abuse handling, and operational security telemetry;
- user-facing pairing approval, trust inspection, revocation, recovery, and key-replacement UX;
- complete GoreeCloud Identity, Wardveil Security, Privacy Shield, Everkeep, and GoreeCloud Mesh production integration evidence;
- deployment and release acceptance.

## Threat classes

The implementation and production design must account for at least:

- unauthorized or stolen devices;
- compromised accounts;
- malicious or untrusted peers;
- pairing, session, record, or deletion replay;
- protocol downgrade attempts;
- TLS peer-key or identity substitution;
- tampering with transferred chunks;
- relay or temporary-storage compromise;
- accidental public service exposure;
- abusive share-link enumeration or download attempts;
- conflict-driven data loss;
- destructive deletion propagation;
- resource exhaustion;
- sensitive information leakage through logs or diagnostics.

## Peer transport security boundary

A valid TLS session proves possession of the pinned device key and encrypts/authenticates the transport. It does **not** by itself grant account, dataset, folder, transfer, or share authorization.

The expected remote identity must be resolved from explicit trust state by higher-level code. A self-signed certificate from an unknown key, a matching device name without the pinned key, LAN presence, NetBird membership, or raw TCP reachability is insufficient.

The current secure wrapper intentionally uses exact key pinning instead of public Web PKI hostname trust because GoreeCloud device authorization is account/device based. The client disables default Web-PKI verification only while replacing it with strict `VerifyConnection` checks for the exact authorized device identity and key.

## Share security

Temporary sharing is intended to use client-side end-to-end encryption. A storage or relay service must not need plaintext file contents to perform its role. Exact key agreement, encryption, metadata-protection, recipient authentication, expiry, and revocation behavior remain subject to separate protocol security review and implementation.

The direct-peer TLS primitive does not by itself satisfy Share's end-to-end encryption requirements for relay-backed delivery.

## Local defaults

Development services bind to loopback by default. Public exposure must be an explicit deployment decision protected by appropriate GoreeCloud publication, firewall, authentication, reverse-proxy, rate-limit, and application-authorization controls.

## Logging

Logs must not include file contents, encryption keys, authentication secrets, private device keys, reusable bearer credentials, or unnecessary full paths. Operational identifiers should be scoped and minimized.

Secure-peer failures should identify the class of failure needed for operations without logging certificate private material, raw secrets, or unnecessary user data.

## Recovery safety

Security controls must not undermine recoverability. Revocation, key rotation, conflict handling, deletion propagation, and future encrypted-session/account recovery must have documented recovery behavior before production approval.

Everkeep remains the continuity and recovery authority; synchronized copies do not become backups merely because transport is authenticated and encrypted.

## Stable security gate

Before Stable, evidence must cover at minimum:

- multi-user isolation and application/folder authorization;
- device enrollment, revocation, recovery, and key replacement;
- secure-session connection admission and revocation behavior;
- TLS downgrade/key-substitution/identity-binding tests;
- replay and stale-operation behavior;
- transfer integrity and interrupted-transfer recovery;
- conflict and deletion safety;
- path traversal and filesystem escape resistance for folder synchronization;
- resource exhaustion, rate limiting, and bounded concurrency;
- logging privacy and secret separation;
- Share E2EE/expiry/revocation behavior where claimed;
- dependency/security review;
- migration, rollback, packaging, deployment, and exact-release validation;
- applicable Wardveil Security, Privacy Shield, Everkeep, GoreeCloud Identity, GoreeCloud Mesh, and Glaze UI acceptance evidence.

## Current claim boundary

GoreeCloud Sync now contains tested source controls for pairing replay/expiry, durable trusted-device authorization, trust-enforced peer resolution, and an authenticated TLS 1.3 direct-peer primitive. The complete product security model, production runtime composition, and Stable security acceptance remain incomplete.
