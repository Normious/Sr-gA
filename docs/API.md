# API reference

Every authenticated endpoint takes JSON and an `X-API-Key` header. This page lists each endpoint with its fields and a tested example. All examples below were executed against the service and return the shapes shown.

## Authentication and conventions

Pass your project key on every call except `GET /health` and `GET /`:

```bash
curl -H "X-API-Key: shop-a-srga-key-2026" http://localhost:4017/stats?days=7
```

Three auth outcomes exist. A missing header returns `401 {"success": false, "error": "Missing X-API-Key header"}`. An unknown or deactivated key returns `401 {"success": false, "error": "Invalid or inactive API Key"}`. A valid key injects the project into request context, so `history` and `stats` automatically scope to your project.

Generation endpoints return XML or plain text with three custom headers:

- `X-Sr-gA-URL-Count`: URLs rendered in the output
- `X-Sr-gA-Cache`: `miss`, `memory`, or `sqlite`
- `X-Sr-gA-Duration-Ms`: generation time, excluding cache-hit fast paths

Error responses always use the envelope `{"success": false, "error": "human-readable reason"}` with a matching HTTP status.

## POST /sitemap/generate

Build a `sitemap.xml` from a URL list. Returns `application/xml`, or `application/gzip` with `compress: true`.

| Field | Type | Required | Notes |
| :--- | :--- | :--- | :--- |
| `urls` | array | yes | 1 to 50000 entries (`MAX_URLS_PER_REQUEST`) |
| `urls[].loc` | string | yes | Must start with `http://` or `https://`, max 2048 chars |
| `urls[].lastmod` | string | no | Passed through verbatim, use `YYYY-MM-DD` |
| `urls[].changefreq` | string | no | Falls back to the project default (`weekly`) |
| `urls[].priority` | number | no | Falls back to the project default (`0.5`), clamped to 0-1 |
| `pretty_print` | bool | no | Indented XML when true, compact otherwise |
| `compress` | bool | no | Gzip the output when true |

```bash
curl -X POST http://localhost:4017/sitemap/generate \
  -H "X-API-Key: shop-a-srga-key-2026" \
  -H "Content-Type: application/json" \
  -d '{"urls": [{"loc": "https://shop-a.com/", "changefreq": "daily", "priority": 1.0},
                {"loc": "https://shop-a.com/about", "changefreq": "monthly", "priority": 0.5}],
       "pretty_print": true}'
```

```xml
<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://shop-a.com/</loc>
    <changefreq>daily</changefreq>
    <priority>1</priority>
  </url>
  <url>
    <loc>https://shop-a.com/about</loc>
    <changefreq>monthly</changefreq>
    <priority>0.5</priority>
  </url>
</urlset>
```

An entry with an empty `loc` is skipped silently. A non-HTTP(S) URL or an over-long URL fails the whole request with `400`.

## POST /sitemap/index

Build a `sitemapindex.xml` that points at other sitemap files. At least one entry is required.

| Field | Type | Required | Notes |
| :--- | :--- | :--- | :--- |
| `sitemaps` | array | yes | One or more `{loc, lastmod}` entries |
| `sitemaps[].loc` | string | yes | URL of a sitemap file |
| `sitemaps[].lastmod` | string | no | Passed through verbatim |
| `pretty_print` | bool | no | Indented XML when true |
| `compress` | bool | no | Gzip the output when true |

```bash
curl -X POST http://localhost:4017/sitemap/index \
  -H "X-API-Key: shop-a-srga-key-2026" \
  -H "Content-Type: application/json" \
  -d '{"sitemaps": [{"loc": "https://shop-a.com/sitemap-products.xml", "lastmod": "2026-09-26"},
                    {"loc": "https://shop-a.com/sitemap-blog.xml"}],
       "pretty_print": true}'
```

```xml
<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap>
    <loc>https://shop-a.com/sitemap-products.xml</loc>
    <lastmod>2026-09-26</lastmod>
  </sitemap>
  <sitemap>
    <loc>https://shop-a.com/sitemap-blog.xml</loc>
  </sitemap>
</sitemapindex>
```

## POST /robots/generate

Build a `robots.txt` from rule blocks. Returns `text/plain`. At least one rule is required, and each rule needs a `user_agent`.

| Field | Type | Required | Notes |
| :--- | :--- | :--- | :--- |
| `rules` | array | yes | One block per user-agent |
| `rules[].user_agent` | string | yes | e.g. `*` or `GPTBot` |
| `rules[].allow` | string array | no | `Allow:` lines |
| `rules[].disallow` | string array | no | `Disallow:` lines |
| `rules[].crawl_delay` | int | no | `Crawl-delay:` line |
| `sitemaps` | string array | no | `Sitemap:` lines |
| `host` | string | no | `Host:` line |
| `include_comments` | bool | no | Header comment crediting Sr-gA |

```bash
curl -X POST http://localhost:4017/robots/generate \
  -H "X-API-Key: shop-a-srga-key-2026" \
  -H "Content-Type: application/json" \
  -d '{"rules": [{"user_agent": "*", "allow": ["/"],
                  "disallow": ["/admin/", "/api/"], "crawl_delay": 10},
                 {"user_agent": "GPTBot", "disallow": ["/"]}],
       "sitemaps": ["https://shop-a.com/sitemap.xml"],
       "host": "shop-a.com", "include_comments": true}'
```

```text
# robots.txt generated by Sr-gA
# https://github.com/Normious/Sr-gA

User-agent: *
Allow: /
Disallow: /admin/
Disallow: /api/
Crawl-delay: 10

User-agent: GPTBot
Disallow: /

Sitemap: https://shop-a.com/sitemap.xml
Host: shop-a.com
```

## POST /urls/validate

Check URLs concurrently over a bounded goroutine pool. Each URL gets a `HEAD` request with `GET` fallback on `405`/`501`.

| Field | Type | Required | Notes |
| :--- | :--- | :--- | :--- |
| `urls` | string array | yes | 1 to 500 entries (`MAX_URLS_FOR_VALIDATION`) |
| `timeout_seconds` | int | no | Per-request timeout, defaults to 10 |
| `concurrency` | int | no | Pool size, defaults to 50 |

```bash
curl -X POST http://localhost:4017/urls/validate \
  -H "X-API-Key: shop-a-srga-key-2026" \
  -H "Content-Type: application/json" \
  -d '{"urls": ["https://example.com/", "https://example.com/missing-page-xyz"],
       "timeout_seconds": 10, "concurrency": 20}'
```

```json
{
  "success": true,
  "total": 2,
  "valid_count": 1,
  "invalid_count": 1,
  "duration_ms": 847,
  "results": [
    {"url": "https://example.com/", "status_code": 200, "response_time_ms": 342, "is_valid": true},
    {"url": "https://example.com/missing-page-xyz", "status_code": 404, "response_time_ms": 289,
     "is_valid": false, "error": "Not Found"}
  ]
}
```

A result counts as valid for any 2xx or 3xx status. `final_url`, `redirect_count`, and `content_type` appear when the check produces them.

## POST /urls/discover

Fetch a published sitemap and return its URLs. Set `include_alternates` to follow a sitemap index recursively up to `max_depth` (default 3). The handler caps the server-side fetch at 60 seconds and 50 MB.

```bash
curl -X POST http://localhost:4017/urls/discover \
  -H "X-API-Key: shop-a-srga-key-2026" \
  -H "Content-Type: application/json" \
  -d '{"sitemap_url": "https://example.com/sitemap.xml",
       "include_alternates": true, "max_depth": 3}'
```

```json
{
  "success": true,
  "sitemap_url": "https://example.com/sitemap.xml",
  "url_count": 2,
  "duration_ms": 16,
  "urls": [
    {"loc": "https://example.com/a", "source": "https://example.com/sitemap.xml"},
    {"loc": "https://example.com/b", "source": "https://example.com/sitemap.xml"}
  ]
}
```

Unreachable hosts and pages without sitemap content return `502` with a `Discovery failed:` reason.

## GET /history

List past operations for your project, newest first. All filters are optional.

| Query param | Notes |
| :--- | :--- |
| `operation` | Filter by `sitemap`, `sitemap_index`, `robots`, `validate`, or `discover` |
| `status` | Filter by `success` or `failed` |
| `search` | Substring match on operation and error message |
| `limit` | Page size, default 20, max 100 |
| `offset` | Page offset, default 0 |

```bash
curl "http://localhost:4017/history?operation=sitemap&limit=5" \
  -H "X-API-Key: shop-a-srga-key-2026"
```

The response wraps `entries` with a `pagination` object carrying `total`, `limit`, and `offset`.

## GET /stats

Aggregate usage over trailing days. `days` defaults to 30 and caps at 365.

```bash
curl "http://localhost:4017/stats?days=7" \
  -H "X-API-Key: shop-a-srga-key-2026"
```

```json
{
  "success": true,
  "days": 7,
  "daily": [{"date": "2026-09-26", "sitemaps_generated": 5, "total_urls": 8,
             "urls_validated": 7, "cache_hits": 1, "failed_count": 0}],
  "totals": {"sitemaps_generated": 5, "total_urls": 8, "urls_validated": 7,
             "cache_hits": 1, "failed_count": 0, "cache_hit_rate": 20},
  "by_operation": {"robots": 1, "sitemap": 2, "sitemap_index": 1, "validate": 1}
}
```

## GET /health and GET /

`GET /health` skips authentication and reports service, version, cache stats, and timestamp for load-balancer checks. `GET /` returns the service description with the endpoint map.

## Error catalog

| Status | When you see it |
| :--- | :--- |
| `400` | Bad JSON, empty URL list, non-HTTP(S) URL, URL over 2048 chars, empty sitemap index, rule without `user_agent`, over-limit batch |
| `401` | Missing or invalid `X-API-Key` |
| `502` | Discovery fetch failed (bad host, non-200 status, no sitemap content) |
| `500` | Gzip failure or history query failure |
