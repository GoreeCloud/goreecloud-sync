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
- Session model foundations.

Current boundaries:

- The transport implementation requires a TLS-authenticated peer with a durable trusted-device fingerprint bound by the application trust composition layer.
- GoreeCloud Sync application composition can revalidate current account/device/key trust before transfer start, before sender source reads, before receiver staging writes, and before returning verified success.
- These trust checks are explicit checkpoints, not instantaneous background revocation.
- A verified transfer proves that the received payload matched the declared manifest. It does not establish backup, archival, recovery, or destination-path authorization.
- The manifest filename is descriptive metadata only. Receivers must independently choose and authorize any eventual filesystem destination and must not interpret the filename as a path.

Future implementation:

- Durable resume persistence and interrupted-transfer recovery.
- Production discovery, address selection, listener/dial, reconnect, and session-lifecycle orchestration.
- Destination-path authorization, folder authorization, filesystem confinement, and complete folder synchronization.
- Transfer progress, cancellation, prioritization, rate limits, quotas, and history.
- Broader transfer replay/freshness policy.
- User-facing Nearby and Glaze UI transfer workflows.
