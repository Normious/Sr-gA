# ADR-0002: `net/http` pattern routing over chi

**Date**: 2026-09-26
**Status**: accepted
**Deciders**: Emmanuel Phiri

## Context

Sr-gA needs eight routes with method separation (`POST /sitemap/generate` vs `GET /stats`). Go 1.22 added method-plus-pattern routing to the standard `ServeMux`, which covers this surface without middleware chains or parameter parsing.

## Decision

We route with stdlib `net/http` (`mux.HandleFunc("POST /sitemap/generate", ...)`) and add no router dependency.

## Alternatives Considered

### Alternative 1: chi
- **Pros**: Familiar middleware ecosystem, path params if routes grow
- **Cons**: Extra dependency for routes that need no params or groups
- **Why not**: Nothing in the route table justifies it

### Alternative 2: gorilla/mux
- **Pros**: Battle-tested, feature-rich
- **Cons**: Archived upstream, heavier than the problem
- **Why not**: Dead project for a new service

## Consequences

### Positive
- Zero router CVEs to track, zero DSL to learn
- Route table reads as plain strings in `cmd/srga/main.go`

### Negative
- If routes ever need path parameters or sub-router groups, expect a migration to chi

### Risks
- None material: the route surface is fixed by the service contract
