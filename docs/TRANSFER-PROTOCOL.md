# GoreeCloud Sync Transfer Protocol

## Purpose

This document defines the initial transfer-engine model.

## Protocol

GC-SYNC/1

## Flow

1. Trusted devices authenticate.
2. Transfer session is created.
3. Files are split into chunks.
4. Chunks are transferred and validated.
5. Final content integrity is verified.

## Guarantees

- Resumable transfers
- Integrity verification
- No silent corruption
- No required cloud relay
