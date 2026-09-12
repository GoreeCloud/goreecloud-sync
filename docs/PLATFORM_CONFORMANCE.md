# Mandatory Native and Platform Conformance

Effective August 24, 2026, this GoreeCloud application must be built and maintained as original GoreeCloud-owned software from the ground up.

Small, technically necessary foundational dependencies remain permitted where independent reimplementation would reduce security, correctness, interoperability, standards compliance, or maintainability. Examples include WireGuard, cryptographic and encryption libraries, protocol libraries, codecs, database engines, operating-system APIs, and comparable critical foundations. Such dependencies must not become the application shell or define the GoreeCloud product identity.

GoreeCloud Sync is one of GoreeCloud's eight Integral Platform Systems. This repository implements the Sync authority itself and must remain current with the authoritative GoreeCloud Platform Contract while preserving the independent authority of the other seven systems.

Every applicable GoreeCloud Sync runtime, service, administration surface, client, protocol adapter, and release boundary must be evaluated against all eight Integral Platform Systems:

1. GoreeCloud Manager
2. Privacy Shield
3. Wardveil Security
4. Everkeep
5. Glaze UI
6. GoreeCloud Mesh
7. GoreeCloud Identity
8. GoreeCloud Sync

The Sync self-relationship identifies this repository as the synchronization, state-coordination, reconciliation, offline-continuity, and cross-device state authority. It does not waive the requirement to implement and accept every applicable cross-system relationship.

Required integrations must be substantive and evidence-backed. In particular:

- GoreeCloud Manager must receive bounded, truthful operational and synchronization-state visibility where applicable without becoming synchronization authority.
- Privacy Shield must authorize applicable synchronization purposes, minimization, retention, disclosure, deletion propagation, and data-use boundaries. Sync must not infer consent or privacy authority from authentication alone.
- Wardveil Security must supply applicable trust, integrity, risk, and protective decisions for devices, sessions, transfers, and synchronized operations. Sync-local cryptographic checks do not replace Wardveil acceptance.
- Everkeep must protect applicable Sync-owned durable configuration, trusted-device relationships, replication/control metadata, recovery, migration, and rollback state. Synchronization must never be represented as backup or recovery evidence by itself.
- Glaze UI must govern user-facing Sync administration and client experiences on every applicable supported form factor.
- GoreeCloud Mesh must provide applicable discovery, reachability, coordination, and transport without becoming the authority for synchronization semantics or data reconciliation.
- GoreeCloud Identity must establish applicable account, device, service, session, credential, authorization, revocation, and delegated-authority boundaries. Sync-local device identity is a Development foundation, not accepted GoreeCloud Identity integration by itself.
- GoreeCloud Sync must own synchronization semantics, version coordination, conflict handling, authorized replication, offline continuity, and reconciliation without taking authority from the originating application or another Integral Platform System.

No release or service state may be classified or retained as Stable or production-ready unless native application qualification and current validated conformance with every applicable Integral Platform System are complete. A missing, materially incomplete, unvalidated, unaccepted, or materially outdated applicable integration is a Stable blocker. Genuine non-applicability must be explicitly justified and supported by evidence rather than silently omitted.

Repository manifests, CI, release documentation, project specifications, user-facing status, and change records must distinguish architecture adoption from runtime acceptance. A passing schema check, source test, interface stub, status card, logo, or documentation statement does not establish production integration.

If this repository contains inherited or upstream-derived application code, that code is transitional or historical. It may be maintained for security, continuity, migration, compatibility, recovery, and rollback while a native replacement is developed, but it is not the approved final architecture.

Repository CI, release documentation, project specifications, and change logs must progressively enforce and record this eight-system contract.
