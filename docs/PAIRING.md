# Pairing foundation

GoreeCloud Sync pairing uses an Ed25519 key-possession proof bound to a canonical device ID and one-time challenge. A valid proof confirms possession of the presented private key; it does not automatically authorize application data access.

Current source also includes:

- cryptographically random short-lived pairing challenges;
- exact challenge consumption with expiry and replay rejection;
- `VerifiedPairing` output only after proof verification and challenge consumption;
- explicit account-scoped trusted-device authorization from verified pairing state;
- durable trusted-device records with explicit revocation and active-key replacement protection;
- trusted-peer checks that require the authenticated device ID and key fingerprint to remain authorized for the account;
- a TLS 1.3 secure-peer primitive that accepts an already-trusted expected device ID and exact raw Ed25519 public key, proves mutual key possession, and binds the later GC-SYNC handshake to those identities.

Pairing proof, durable device trust, transport authentication, and application authorization remain separate boundaries. A successful pairing proof or secure TLS connection alone does not grant dataset, folder, transfer, share, or administrative permission.

Production account/trust lookup orchestration, user-facing approval/recovery/key-replacement UX, revocation-aware reconnect behavior, rate limiting/security events, and Stable acceptance remain pending.
