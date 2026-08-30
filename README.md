# GoreeCloud Sync

GoreeCloud Sync is a privacy-first, self-hosted synchronization and secure-transfer platform for GoreeCloud.

The product is organized around three related modes:

- **Sync** — persistent replication of explicitly approved application datasets and, later, approved folders between authorized endpoints.
- **Nearby** — direct local or private-network transfer of files, folders, text, URLs, and supported payloads.
- **Share** — temporary end-to-end encrypted delivery, including expiring links and direct peer delivery when available.

The product direction is informed by useful capabilities in projects such as Syncthing, LocalSend, and wormhole.app while remaining an independently designed GoreeCloud product.

## Project status

**Lifecycle:** Active Development — pre-Stable.

The repository has advanced beyond its original Milestone 0 shell. Current source foundations include:

- the Go service and CLI shell with loopback-safe development defaults and bounded shutdown behavior;
- pre-stabilization `GC-SYNC/1` control framing and TCP peer-transport helpers;
- Ed25519 device identity and signed pairing-proof primitives;
- short-lived one-time pairing challenges with exact consumption, expiry rejection, and replay rejection;
- durable account-scoped trusted-device authorization with explicit revocation, active-key replacement protection, and stored key/fingerprint validation;
- fail-closed trusted peer resolution that requires the authenticated device ID and key fingerprint to remain authorized for the configured account;
- first-party dataset capability negotiation for GoreeCloud Browser, Search, and Bookmarks;
- transport-neutral versioned record envelopes, deterministic conflict resolution, and payload-free tombstones;
- authenticated peer/session boundaries plus Privacy Shield and Wardveil decision interfaces before durable record acceptance;
- persistent replication stores and ingestion/retrieval handlers for `search.history`, `bookmarks.items`, `browser.tabs`, and `browser.history`;
- record-bound proof, replay/high-water, observation-receipt, tombstone-convergence, and protected device-key lifecycle foundations;
- bounded authenticated retrieval with deterministic record-ID cursor pagination.

These are source and development contracts. They do **not** establish a complete production synchronization product. The default development service is not a production deployment, and Sync routes are registered only when the service is constructed with the required replication ingestor and authenticated peer resolver. Durable trusted-device enforcement is a source boundary that must also be wired into the applicable runtime composition; it does not substitute for authenticated encrypted peer-session establishment.

No Stable or production-readiness claim should be inferred from repository source, passing CI, or the presence of these foundations.

## Governing boundaries

GoreeCloud Sync follows several non-negotiable design rules:

- Synchronization is not a backup system; Everkeep remains the platform continuity, preservation, backup, and recovery authority.
- A synchronized duplicate is not automatically a recoverable backup.
- Every user, device, dataset, folder, and share must be explicitly authorized for the applicable operation.
- Network connectivity does not imply application authorization.
- Application ownership boundaries remain intact: Browser, Search, and Bookmarks own their record semantics while Sync coordinates authorized replication.
- Privacy Shield purpose/consent decisions and Wardveil trust decisions are server-side acceptance gates rather than client assertions.
- Deleted application payload must not be retained merely to communicate deletion; synchronized tombstones are payload-free.
- Live application-managed databases and consistency-sensitive internal state are not ordinary sync targets.
- Conflicts must fail safely and remain visible rather than being silently discarded.
- Temporary sharing must be encrypted end to end where that capability is claimed.
- No advertising, behavioral tracking, or mandatory hosted GoreeCloud account is part of the product model.
- Production secrets, bearer sessions, device private keys, and private transfer contents must not be stored in this repository.

## Development service

The default HTTP listener binds to `127.0.0.1:8787` to avoid accidental public exposure during development.

Always-available development endpoints:

- `GET /healthz`
- `GET /api/v1/status`

When the server is explicitly constructed with a replication ingestor and authenticated peer resolver, source handlers are available for:

- `POST` and `GET /api/v1/sync/search/history`
- `POST` and `GET /api/v1/sync/bookmarks/items`
- `POST` and `GET /api/v1/sync/browser/tabs`
- `POST` and `GET /api/v1/sync/browser/history`

Authenticated retrieval is record-ID ordered and supports bounded cursor pagination with `limit` and exclusive `after` query parameters. The current server default is 256 records per page, the maximum accepted page size is 1,024 records, and Sync record IDs are bounded to 512 bytes so an accepted record can always be represented by the continuation contract. Retrieval scans and validates the complete persisted dataset before returning a page so off-page corruption remains fail-closed, while retaining only a page-bounded candidate window instead of materializing every persisted envelope in memory.

Run the current development checks locally:

```bash
go test ./...
go vet ./...
go build ./cmd/goreecloud-sync
go run ./cmd/goreecloud-sync serve
```

Then inspect the base development service:

```bash
curl http://127.0.0.1:8787/healthz
curl http://127.0.0.1:8787/api/v1/status
```

## Still incomplete

Major production and product work remains, including reviewed authenticated encryption for peer sessions, transport/session replay protection beyond the implemented one-time pairing challenge, production account/runtime wiring, user-facing pairing approval and revocation UX, LAN discovery, complete resumable file/folder synchronization, Nearby, Share E2EE, complete multi-user authorization and administration, Glaze UI product surfaces, Android and Debian clients, migration/rollback tooling, monitoring, deployment, and full Glaze UI/Wardveil Security/Privacy Shield/Everkeep/GoreeCloud Mesh/GoreeCloud Identity acceptance evidence appropriate to the final scope.

Syncthing or any other existing service must not be considered replaced merely because GoreeCloud Sync source foundations exist.

## Repository areas

```text
cmd/                    Command-line entry points
internal/               Private Go implementation packages
protocol/               Pre-stabilization protocol contracts and records
docs/                   Architecture and engineering documentation
web/                    Future Glaze UI web administration client
android/                Future native Android client
packaging/              Future Debian and other release packaging
tests/                  Cross-component acceptance and protocol tests
.github/workflows/      Continuous integration and release automation
```

Directories are added when implementation work reaches them rather than as empty placeholders.

## Product documentation

- [Specifications](SPECIFICATIONS.md)
- [Competitive objectives](COMPETITIVE-OBJECTIVES.md)
- [Features](FEATURES.md)
- [Benefits](BENEFITS.md)
- [Architecture](docs/architecture.md)
- [Protocol direction](protocol/README.md)
- [Security model](docs/security-model.md)
- [Security policy](SECURITY.md)
- [Contributing](CONTRIBUTING.md)

## License

GoreeCloud Sync is licensed under the GNU Affero General Public License v3.0. See [LICENSE](LICENSE).
