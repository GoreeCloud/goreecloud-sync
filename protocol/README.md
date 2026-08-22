# GoreeCloud Sync Protocol

## Status

Pre-stabilization design area. No wire protocol is stable, versioned for compatibility, or approved for production use yet.

## Goals

The protocol layer will support:

- authenticated device identity;
- explicit user and folder authorization;
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
- silently discard file conflicts;
- treat byte delivery as proof of recoverability;
- require a permanently hosted GoreeCloud cloud account;
- depend on one transport provider or relay operator.

## Planned protocol layers

```text
Application operation
  Sync | Nearby | Share
        |
Authorization and capability negotiation
        |
Transfer manifest and integrity contract
        |
Chunk transport and resume state
        |
Authenticated encrypted session
        |
LAN | private network | direct Internet | relay
```

## Identity

Each enrolled device will have an independently revocable cryptographic identity. Device identity is separate from account identity, and a valid device identity is not sufficient to authorize access to a folder or share.

## Transfer manifests

A transfer manifest is expected to describe the minimum information required to verify and resume an operation, including version, transfer identifier, payload type, size information, chunking information, integrity identifiers, and authorized operation. Metadata exposure must be minimized.

## Integrity

Chunk verification and final payload verification will be separate states. A completed network transfer cannot be reported as verified until the applicable integrity checks succeed.

The exact hashing algorithm, authenticated-encryption construction, key agreement, and key derivation choices remain security-review decisions and are intentionally not frozen by this Milestone 0 document.

## Compatibility

Protocol changes must use explicit version/capability negotiation. Breaking changes must not be shipped under an unchanged protocol version.

## Security review gate

No temporary-share E2EE claim or stable sync protocol claim may be made until the concrete protocol specification, threat model, cryptographic dependency choices, test vectors, downgrade behavior, replay protections, and revocation semantics have been reviewed and validated.
