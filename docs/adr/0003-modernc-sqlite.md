# ADR-0003: `modernc.org/sqlite` over `mattn/go-sqlite3`

**Date**: 2026-09-26
**Status**: accepted
**Deciders**: Emmanuel Phiri

## Context

Sr-gA persists tenants, history, cache, and rollups in SQLite. The Dockerfile targets `FROM scratch` for a ~10 MB image, which forbids any C dependency at build or runtime.

## Decision

We use `modernc.org/sqlite`, a pure-Go SQLite port, so `CGO_ENABLED=0` builds succeed.

## Alternatives Considered

### Alternative 1: `mattn/go-sqlite3`
- **Pros**: Most widely used, thin wrapper over battle-tested C SQLite
- **Cons**: Requires CGO, which forces a libc base image and kills `FROM scratch`
- **Why not**: Breaks the single-static-binary deployment goal

### Alternative 2: Postgres sidecar
- **Pros**: Real concurrent writer story, familiar ops
- **Cons**: Second container, migrations service, credentials, backups for a single-file workload
- **Why not**: Order-of-magnitude more ops for no query need beyond key lookups and rollups

## Consequences

### Positive
- Static binary, scratch image, WAL mode with foreign keys on
- `go build` works on any host with no C toolchain

### Negative
- Larger `go.sum` and slower first compile (the pure-Go port is big)
- Binds us to the modernc fork's SQLite version cadence

### Risks
- Performance edge cases in the Go port under heavy concurrent writes; mitigated by capping the pool at 10 connections and keeping writes small and indexed
