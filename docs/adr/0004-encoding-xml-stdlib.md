# ADR-0004: `encoding/xml` for sitemap output

**Date**: 2026-09-26
**Status**: accepted
**Deciders**: Emmanuel Phiri

## Context

Sitemap output must be spec-compliant XML (`urlset`, `sitemapindex`) with an XML declaration header. Struct tags map the domain model to elements directly, so the marshaler choice shapes every response.

## Decision

We marshal with stdlib `encoding/xml` and prepend the `<?xml version="1.0" encoding="UTF-8"?>` header manually, since `xml.Marshal` omits it.

## Alternatives Considered

### Alternative 1: JSON-to-XML conversion library
- **Pros**: Lets handlers stay in JSON-shaped structs
- **Cons**: Extra dep, lossy namespace control, harder to audit byte output
- **Why not**: Struct tags already express the schema exactly

### Alternative 2: String-template XML
- **Pros**: No marshaling surprises, full byte control
- **Cons**: Manual escaping bugs, the classic injection vector for sitemap poisoning via crafted URLs
- **Why not**: The stdlib escapes correctly and the template buys nothing

## Consequences

### Positive
- Zero-dep marshaling, output audited by unit test against the spec shape
- `priority` always emitted and clamped to 0-1; URLs over 2048 chars rejected before render

### Negative
- Manual header prepend is a wart every reader must notice once

### Risks
- None material: output is covered by `sitemap_test.go` byte assertions
