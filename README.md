# GoreeCloud Sync

GoreeCloud Sync is a privacy-first, self-hosted multi-user synchronization and secure-transfer platform for GoreeCloud.

The project is designed around three related modes:

- **Sync** — persistent replication of explicitly approved folders between authorized endpoints.
- **Nearby** — direct local or private-network transfer of files, folders, text, URLs, and supported payloads.
- **Share** — temporary end-to-end encrypted delivery, including expiring links and direct peer delivery when available.

The product direction is inspired by the useful capabilities of Syncthing, LocalSend, and wormhole.app while remaining an independently designed GoreeCloud product.

## Project status

**Lifecycle:** Development — Milestone 0 foundation.

The repository currently contains the initial service and CLI shell, governance documents, architecture records, security boundaries, and CI. Persistent synchronization, discovery, transfer protocols, multi-user persistence, encrypted shares, Android clients, Debian packaging, and production deployment are not yet implemented.

No Stable or production-readiness claim should be inferred from the existence of this repository.

## Governing boundaries

GoreeCloud Sync follows several non-negotiable design rules:

- Synchronization is not a backup system.
- A synchronized duplicate is not automatically a recoverable backup.
- Every user, device, folder, and share must be explicitly authorized.
- Network connectivity does not imply application authorization.
- Live application-managed databases and consistency-sensitive internal state are not ordinary sync targets.
- Conflicts must fail safely and remain visible rather than being silently discarded.
- Temporary sharing must be encrypted end to end where that capability is claimed.
- No advertising, behavioral tracking, or mandatory hosted GoreeCloud account is part of the product model.
- Production secrets, device private keys, and private transfer contents must not be stored in this repository.

## Initial architecture

The first implementation uses Go for the service and CLI foundation. The default HTTP listener binds to `127.0.0.1:8787` to avoid accidental public exposure during development.

Current development endpoints:

- `GET /healthz`
- `GET /api/v1/status`

Run locally:

```bash
go test ./...
go run ./cmd/goreecloud-sync serve
```

Then inspect:

```bash
curl http://127.0.0.1:8787/healthz
curl http://127.0.0.1:8787/api/v1/status
```

## Planned repository areas

```text
cmd/                    Command-line entry points
internal/               Private Go implementation packages
protocol/               Protocol contracts and compatibility records
docs/                   Architecture and engineering documentation
web/                    Future Glaze UI web administration client
android/                Future native Android client
packaging/              Future Debian and other release packaging
tests/                  Cross-component acceptance and protocol tests
.github/workflows/      Continuous integration and release automation
```

Directories are added when implementation work reaches them rather than as empty placeholders.

## Product documentation

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
