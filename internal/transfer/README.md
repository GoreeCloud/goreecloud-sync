# Transfer Engine

The transfer engine provides Development-stage payload modeling and integrity contracts for GoreeCloud Sync.

Current capabilities:

- Bounded transfer chunk sizing with a current 1 MiB maximum.
- SHA-256 hashing primitives for chunks and complete payloads.
- Versioned single-file manifests with ordered chunk metadata and structural validation.
- Manifest filenames are path-independent leaf metadata; path separators plus `.` and `..` are rejected and never grant destination authority.
- Per-chunk and whole-payload integrity verification.
- Versioned one-to-one file/text offer, decision, completion, and verified-receipt control records.
- Cryptographically random 128-bit transfer identifiers.
- Integration with the authenticated peer transport for bounded encrypted chunk movement.
- Explicit receiver authorization before payload bytes are accepted for transfer.
- Staging semantics that require callers to publish received content only after verified success.
- A local publication service that independently validates the offer/receipt binding, requires caller-supplied destination authorization, re-verifies staged content against the manifest, confines publication to a canonical caller-selected directory, and refuses overwrite of an existing destination.
- Session model foundations.

Current boundaries:

- The transport implementation requires a TLS-authenticated peer with a durable trusted-device fingerprint bound by the application trust composition layer.
- GoreeCloud Sync application composition can revalidate current account/device/key trust before transfer start, before sender source reads, before receiver staging writes, and before returning verified success.
- These trust checks are explicit checkpoints, not instantaneous background revocation.
- A verified transfer proves that the received payload matched the declared manifest. It does not establish backup, archival, recovery, or destination authorization.
- The manifest filename is descriptive leaf metadata only. A remote peer never selects the local publication root.
- Local publication requires a separate opaque destination identity and authorization callback supplied by trusted local application composition. The current source service does not itself implement GoreeCloud Identity policy lookup, user-facing destination selection, folder authorization, conflict-resolution UX, or production storage policy.
- Local publication is no-overwrite: an existing destination is preserved rather than silently replaced.

Future implementation:

- Durable resume persistence and interrupted-transfer recovery.
- Production discovery, address selection, listener/dial, reconnect, and session-lifecycle orchestration.
- Production destination-authority adapters, folder authorization, broader filesystem policy, and complete folder synchronization.
- Transfer progress, cancellation, prioritization, rate limits, quotas, and history.
- Broader transfer replay/freshness policy.
- User-facing Nearby and Glaze UI transfer workflows.
