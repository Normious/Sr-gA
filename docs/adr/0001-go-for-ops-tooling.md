# ADR-0001: Go for the service language

**Date**: 2026-09-26
**Status**: accepted
**Deciders**: Emmanuel Phiri

## Context

Sr-gA is ops tooling: generate sitemaps on demand, validate hundreds of URLs fast, and deploy as cheaply as possible. The workload is I/O-bound fan-out plus XML marshaling, and the service must run on any server with zero runtime setup.

## Decision

We write Sr-gA in Go 1.22+, compiling to one static binary with no runtime.

## Alternatives Considered

### Alternative 1: Node.js
- **Pros**: Fast to prototype, large ecosystem
- **Cons**: 150 MB+ image, slower bulk XML generation, promise-pool tuning for fan-out
- **Why not**: Heavier deployment for a tool whose whole point is lightness

### Alternative 2: Python
- **Pros**: Simplest XML and HTTP code
- **Cons**: 5-10s for a 50k-URL sitemap vs 100-200 ms in Go, interpreter on every host
- **Why not**: Too slow at the top end of the workload

### Alternative 3: Rust
- **Pros**: 5 MB binary, excellent concurrency via Tokio
- **Cons**: Heavier async setup for this fan-out shape, longer builds, smaller hiring overlap with the rest of the stack
- **Why not**: Go clears the bar with less machinery

## Consequences

### Positive
- One `scp`-able binary, `FROM scratch` image at 10.1 MB
- Goroutine pool validates 500 URLs in under 3 seconds
- Standard-library HTTP, XML, and structured logging with two external deps total

### Negative
- No generics-heavy abstractions; boilerplate stays verbose in places
- Contributors need Go toolchain familiarity

### Risks
- Go release drift: module targets 1.22 pattern routing, older toolchains fail to build; pinned Dockerfile base (`golang:1.22-alpine`) mitigates this
