# Security and Privacy Model

## Status

Milestone 0 design record. Security properties described as planned are not implementation claims.

## Trust boundaries

GoreeCloud Sync treats users, devices, network paths, relay infrastructure, temporary storage, application APIs, and synchronized folders as separate trust boundaries.

Key principles:

- authenticate every device independently;
- authorize every data relationship explicitly;
- use least privilege for folder and transfer access;
- keep network authorization separate from application authorization;
- make revocation a first-class lifecycle operation;
- minimize stored metadata and logs;
- use established cryptographic libraries rather than custom primitives;
- keep production secrets outside source control.

## Planned threat classes

The implementation must account for at least:

- unauthorized or stolen devices;
- compromised accounts;
- malicious or untrusted peers;
- replay and downgrade attempts;
- tampering with transferred chunks;
- relay or temporary-storage compromise;
- accidental public service exposure;
- abusive share-link enumeration or download attempts;
- conflict-driven data loss;
- destructive deletion propagation;
- sensitive information leakage through logs or diagnostics.

## Share security

Temporary sharing is intended to use client-side end-to-end encryption. A storage or relay service must not need plaintext file contents to perform its role. Exact key agreement, encryption, and metadata-protection behavior remains subject to protocol security review.

## Local defaults

Development services bind to loopback by default. Public exposure must be an explicit deployment decision protected by the appropriate GoreeCloud publication, firewall, authentication, and reverse-proxy controls.

## Logging

Logs must not include file contents, encryption keys, authentication secrets, private device keys, or unnecessary full paths. Operational identifiers should be scoped and minimized.

## Recovery safety

Security controls must not undermine recoverability. Revocation, key rotation, conflict handling, and deletion propagation must have documented recovery behavior before production approval.
