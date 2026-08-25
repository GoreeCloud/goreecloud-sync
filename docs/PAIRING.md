# Pairing foundation

GoreeCloud Sync pairing uses an Ed25519 key-possession proof bound to a device ID and one-time challenge. A valid proof confirms possession of the presented private key; it does not automatically authorize the device, create a trusted relationship, or establish an encrypted session.

The higher pairing layer must require explicit trust approval, persist the approved device ID and verified public-key fingerprint, reject replayed or expired challenges, and use the approved identity when authenticated transport is introduced.
