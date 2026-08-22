# Security Policy

## Project maturity

GoreeCloud Sync is under active development and has not reached a Stable or production-ready release. Security-sensitive protocol details are not yet frozen.

## Reporting a vulnerability

Please report security vulnerabilities privately through GitHub Security Advisories for this repository when available. Do not publish exploitable details in a public issue before a coordinated fix can be prepared.

A useful report includes:

- affected revision or version;
- affected component;
- reproducible steps or proof of concept;
- expected and observed behavior;
- security impact;
- suggested mitigation if known.

Do not include real user files, credentials, private keys, tokens, or unrelated personal information in a report.

## Security principles

The project intends to maintain:

- explicit user, device, folder, and share authorization;
- independently revocable device identity;
- encrypted authenticated transport;
- end-to-end encryption for temporary shares where claimed;
- least privilege;
- secure local defaults;
- minimal public exposure;
- privacy-conscious logs and diagnostics;
- dependency and vulnerability management;
- recovery-aware conflict and deletion behavior.

## Cryptography

GoreeCloud Sync will use established cryptographic libraries and protocols. The project will not create custom cryptographic primitives. Concrete algorithms and protocol constructions require dedicated review and validation before stabilization.

## Secrets

Production secrets, passwords, API tokens, device private keys, signing secrets, encryption keys, private transfer contents, and recovery secrets must never be committed to this repository.
