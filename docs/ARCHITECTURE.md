# Architecture

Sr-gA is one stateless Go binary plus one SQLite file. All request state lives in that file, so horizontal scaling means one instance per volume, and backups mean copying `srga.db`. This page maps the system boundary, the code inside the binary, the two request flows that matter, and the schema.

## System context

Client services call Sr-gA over HTTP with an API key. Sr-gA answers from its own database and, only for validation and discovery, makes outbound requests to the public internet.

```mermaid
graph LR
    ShopA[Shop A nightly cron] --> SrGA
    Gig[Gig4Gig CMS] --> SrGA
    Blog[Emerge Fund blog] --> SrGA
    SrGA[Sr-gA :4017] --> DB[(srga.db on volume)]
    SrGA --> Net[Public URLs<br/>HEAD/GET for validation<br/>GET for discovery]
```

Sr-gA never calls back into your services. The only network traffic it initiates is the URL checking you explicitly request.

## Inside the binary

Requests pass through three middleware layers, then an auth gate, then one handler per endpoint group. Handlers call the `sitemap` domain package for output, the `cache` package for fast recall, and the `db` package for persistence.

```mermaid
graph TB
    Mux[net/http ServeMux<br/>Go 1.22 pattern routing] --> MW[Recovery → CORS → Logging]
    MW --> Health[GET /health, GET /]
    MW --> Auth[X-API-Key middleware]
    Auth --> H1[sitemap.go<br/>generate]
    Auth --> H2[sitemapindex.go<br/>index]
    Auth --> H3[robots.go<br/>robots]
    Auth --> H4[validate.go<br/>validate + discover]
    Auth --> H5[history.go<br/>history + stats]
    H1 --> Dom[sitemap package<br/>model, generator, robots,<br/>validator, discover]
    H2 --> Dom
    H3 --> Dom
    H4 --> Dom
    H1 --> Mem[cache.MemoryCache<br/>map + RWMutex + TTL]
    H2 --> Mem
    H3 --> Mem
    H1 --> DB[(db package<br/>modernc.org/sqlite, WAL mode)]
    H2 --> DB
    H3 --> DB
    H4 --> DB
    H5 --> DB
```

Package responsibilities stay narrow on purpose:

| Package | Responsibility | Key files |
| :--- | :--- | :--- |
| `config` | Read env with defaults, configure JSON logging | `config.go` |
| `db` | Open SQLite, run `migrations/0001_init.sql`, seed demo projects | `db.go`, `queries.go` |
| `cache` | Thread-safe in-memory map with TTL and capacity eviction | `memory.go` |
| `auth` | Look up the project by `X-API-Key`, stash it in request context | `auth.go` |
| `sitemap` | Pure output logic with no HTTP or SQL imports | `model.go`, `generator.go`, `robots.go`, `validator.go`, `discover.go` |
| `handlers` | Parse JSON, enforce limits, orchestrate cache and DB, write responses | `sitemap.go`, `sitemapindex.go`, `robots.go`, `validate.go`, `history.go`, `health.go` |
| `middleware` | Panic recovery, CORS headers, per-request slog lines | `middleware.go` |

The `sitemap` package imports nothing from `handlers`, `db`, or `cache`, which keeps generation logic unit-testable without a database.

## Generate flow with two-layer cache

Cache keys derive from the SHA-256 hash of the canonical request JSON, so byte-identical requests share entries across projects and restarts. Memory answers in microseconds; SQLite answers after restarts when memory is cold.

```mermaid
sequenceDiagram
    participant C as Client
    participant H as Handler
    participant M as Memory cache
    participant S as SQLite cache
    participant G as Generator
    C->>H: POST /sitemap/generate + JSON
    H->>M: Get(sha256(request))
    alt memory hit
        M-->>H: bytes
        H->>H: LogHistory(cache_hit=memory)
        H-->>C: 200 XML, X-Sr-gA-Cache: memory
    else memory miss
        H->>S: GetCache(hash)
        alt sqlite hit
            S-->>H: row
            H->>M: Put(hash, row)
            H->>H: LogHistory(cache_hit=sqlite)
            H-->>C: 200 XML, X-Sr-gA-Cache: sqlite
        else full miss
            H->>G: GenerateURLSet(...)
            G-->>H: bytes
            H->>M: Put(hash, bytes)
            H->>S: PutCache(hash, ttl)
            H->>H: LogHistory(cache_hit=miss)
            H-->>C: 200 XML, X-Sr-gA-Cache: miss
        end
    end
```

Every branch, including cache hits, writes a `generation_history` row and upserts `daily_summary`, so stats stay accurate when traffic is mostly cached.

## Validate flow

Validation fans out over a semaphore-bounded goroutine pool. Each URL gets a `HEAD` request first with fallback to `GET` on `405` or `501`, which keeps checks light against servers that reject `HEAD`.

```mermaid
sequenceDiagram
    participant C as Client
    participant H as Handler
    participant P as Goroutine pool (N = concurrency)
    participant W as Remote URL
    C->>H: POST /urls/validate {urls, concurrency}
    H->>P: one slot per URL, at most N running
    P->>W: HEAD url (User-Agent: Sr-gA-Validator)
    alt 405 or 501
        P->>W: GET url
    end
    W-->>P: status, headers, final URL
    P-->>H: ValidationResult per URL
    H->>H: count valid vs invalid, LogHistory
    H-->>C: 200 JSON summary + per-URL rows
```

There is no cache on this path: validation results depend on live remote state, so each call re-checks. The `url_validations` table exists in the schema for short-lived result storage if a future endpoint needs it.

## Data model

Five tables cover tenants, audit trail, cache, and rollups. Timestamps are Unix milliseconds; booleans are `0`/`1` integers per SQLite convention.

```mermaid
erDiagram
    projects ||--o{ generation_history : has
    projects ||--o{ daily_summary : rolls_up
    projects ||--o{ url_validations : checks
    projects {
        int id PK
        string name
        string api_key UK
        string default_change_freq
        float default_priority
        int max_urls_per_sitemap
        int validate_urls_default
        int is_active
    }
    generation_history {
        int id PK
        int project_id FK
        string operation
        int url_count
        int valid_count
        int invalid_count
        int output_size_bytes
        int compressed
        string cache_hit
        int duration_ms
        string status
    }
    sitemap_cache {
        int id PK
        string content_hash UK
        string operation
        string output_xml
        int compressed
        int expires_at
    }
    daily_summary {
        int id PK
        int project_id FK
        string date UK
        int sitemaps_generated
        int total_urls
        int urls_validated
        int cache_hits
        int failed_count
    }
```

`sitemap_cache` rows expire by `expires_at` and a background goroutine purges them every 30 minutes. `daily_summary` uses an upsert on `(project_id, date)` so concurrent requests never double-count a day.

## Cross-cutting behavior

Three decisions shape every endpoint. Graceful shutdown drains in-flight requests for up to 10 seconds on `SIGINT`/`SIGTERM`. Structured JSON logs come from `log/slog` with method, path, and duration on each line. SQLite runs in WAL mode with foreign keys on, capped at 10 open connections, which is plenty for an ops tool with bursty traffic.
