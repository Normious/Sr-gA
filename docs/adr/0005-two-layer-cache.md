# ADR-0005: Memory plus SQLite cache by content hash

**Date**: 2026-09-26
**Status**: accepted
**Deciders**: Emmanuel Phiri

## Context

Nightly cron jobs regenerate identical sitemaps, and dashboards re-request robots output. Regeneration is cheap (milliseconds) but pointless when the bytes already exist, and restarts wipe any in-process cache.

## Decision

We cache by SHA-256 of the canonical request JSON in two layers: an in-memory map with TTL and capacity cap, backed by a `sitemap_cache` SQLite table with `expires_at`. Lookups try memory, then SQLite, then generate, and every hit is logged so stats stay honest.

## Alternatives Considered

### Alternative 1: Memory only
- **Pros**: Simplest code, microsecond hits
- **Cons**: Cold after every restart or redeploy, exactly when crons stampede
- **Why not**: Loses the most valuable hits

### Alternative 2: SQLite only
- **Pros**: Persistent, one code path
- **Cons**: Disk read plus deserialization on every repeated request
- **Why not**: Slower hot path for the common cron-repeat case

### Alternative 3: Redis sidecar
- **Pros**: Shared across future replicas, real eviction policies
- **Cons**: Second process to deploy, monitor, and back up for a cache SQLite already covers
- **Why not**: Same overkill verdict as Postgres in ADR-0003

## Consequences

### Positive
- Hot requests answer from memory; post-restart repeats answer from SQLite (`X-Sr-gA-Cache: sqlite`, verified in testing)
- Cache-hit rows still feed `daily_summary`, so hit rate is a real metric, not a gap

### Negative
- Two TTLs to reason about (`MEMORY_CACHE_TTL_SECONDS`, `SQLITE_CACHE_TTL_SECONDS`)
- Hash keys assume canonical JSON; field-order changes in the client produce misses, never wrong hits

### Risks
- Unbounded growth if the purger stalls: the 30-minute `PurgeExpiredCache` goroutine mitigates it, and expiry is checked on read regardless
