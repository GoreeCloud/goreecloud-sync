# Architecture

## Status

Milestone 0 architecture record. This document defines intended boundaries; it does not claim that the full synchronization or transfer system is implemented.

## Product model

GoreeCloud Sync combines three separate workflows behind one authorization and transfer platform:

1. **Sync** — durable folder replication.
2. **Nearby** — immediate device-to-device transfer.
3. **Share** — temporary encrypted delivery.

These modes share device identity, user authorization, integrity verification, transport selection, observability, and policy infrastructure but must remain independently controllable.

## Planned logical components

```text
Clients
  |
  +-- CLI
  +-- Web administration / Glaze UI
  +-- Android
  +-- future platform clients
  |
Application API
  |
  +-- Identity and authorization
  +-- Device trust
  +-- Folder policy
  +-- Transfer coordination
  +-- Share lifecycle
  +-- Conflict management
  +-- Audit/event boundary
  |
Transfer engine
  |
  +-- discovery
  +-- connection negotiation
  +-- chunking and resume
  +-- integrity validation
  +-- direct transport
  +-- relay fallback
  |
Storage adapters
```

## Initial service boundary

The Milestone 0 Go service currently exposes only development health/status endpoints. It intentionally binds to loopback by default and has no file transfer or filesystem write authority.

## Authorization model

The planned authorization hierarchy is:

```text
Person -> Account -> Device -> Explicit capability grants
                         |
                         +-- folder grants
                         +-- transfer grants
                         +-- share grants
```

A network route, LAN discovery event, NetBird membership, or successful TCP/QUIC connection must never grant data access by itself.

Planned folder permissions include read, write, contribute, receive-only, drop-only, and administrative management. Exact permission semantics must be versioned before protocol stabilization.

## Transport preference

For approved endpoints, the expected preference is:

1. direct local connection;
2. direct approved private-network connection;
3. safe direct Internet peer connection when explicitly supported;
4. self-hosted rendezvous or relay fallback.

Raw synchronization ports must not be exposed publicly merely for convenience.

## Data and recovery boundary

Synchronization provides movement and availability. Everkeep, GoreeCloud Backup, filesystem snapshots, and other independent recovery systems provide recovery authority.

Live databases, active application-managed volumes, certificate stores, backup repositories, and other consistency-sensitive state require application-aware protection rather than ordinary file synchronization.

## Technology direction

- Go: service, CLI, concurrency-heavy transfer engine, APIs.
- TypeScript: future Glaze UI web administration client.
- Kotlin: future Android client and background transfer services.
- Rust: selective security-sensitive or performance-critical components when justified.
- SQL: relational state when durable multi-user metadata requires it.

Technology choices remain revisable when implementation evidence shows a safer or simpler option.
