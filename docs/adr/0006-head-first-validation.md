# ADR-0006: HEAD-first URL validation with GET fallback

**Date**: 2026-09-26
**Status**: accepted
**Deciders**: Emmanuel Phiri

## Context

Pre-publish checks must confirm hundreds of URLs quickly without downloading bodies. `HEAD` returns status and headers with minimal bytes, but some servers answer `405`/`501` to `HEAD` even though the page exists.

## Decision

The validator sends `HEAD` first and retries with `GET` only on `405 Method Not Allowed` or `501 Not Implemented`. Statuses 200-399 count as valid, and a semaphore-bounded goroutine pool caps concurrency.

## Alternatives Considered

### Alternative 1: GET-only
- **Pros**: One code path, works against every server
- **Cons**: Downloads full bodies for 500 URLs per batch, slower and ruder to targets
- **Why not**: Wasteful by default when `HEAD` succeeds almost everywhere

### Alternative 2: HEAD-only, no fallback
- **Pros**: Lightest possible check
- **Cons**: False negatives for valid pages on `HEAD`-rejecting servers, the exact failure that erodes trust in the tool
- **Why not**: Correctness beats elegance on a validation endpoint

## Consequences

### Positive
- Minimal bytes in the common case, correct verdicts in the edge case
- Redirect cap (5) and per-request timeout (10s default) bound the worst case

### Negative
- Two request shapes to test instead of one

### Risks
- Servers that rate-limit by User-Agent could throttle bulk checks; the `VALIDATION_USER_AGENT` identifies traffic and concurrency is tunable per request
