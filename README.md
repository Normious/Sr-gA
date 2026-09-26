# Sr-gA: sitemap and robots.txt generator API

Sr-gA is a single-binary Go service that generates spec-compliant `sitemap.xml` files, sitemap indexes, and `robots.txt` files, with concurrent URL validation and usage analytics. You send it URLs as JSON, it returns XML or plain text, and it records every operation per API key. The Docker image is 10.1 MB and starts in about 30 ms.

## What it does

Sr-gA exposes eight endpoints behind per-project API keys:

- `POST /sitemap/generate`: build a `sitemap.xml` from a URL list, with optional gzip output
- `POST /sitemap/index`: build a `sitemapindex.xml` that references multiple sitemap files
- `POST /robots/generate`: build a `robots.txt` from user-agent rules, sitemap URLs, and host
- `POST /urls/validate`: check up to 500 URLs concurrently and report status codes
- `POST /urls/discover`: import URLs by fetching an existing sitemap, following indexes
- `GET /history`: paged log of past operations for your project
- `GET /stats`: daily rollups, totals, cache hit rate, and per-operation counts
- `GET /health`: unauthenticated liveness check with cache stats

Results pass through a two-layer cache. Identical requests return `X-Sr-gA-Cache: memory` or `X-Sr-gA-Cache: sqlite` instead of regenerating output. See [ARCHITECTURE](docs/ARCHITECTURE.md) for the cache design and [API reference](docs/API.md) for every field.

## Run it in 60 seconds

You need Go 1.22 or newer. Clone the repo, copy the example environment file, and start the server:

```bash
git clone https://github.com/Normious/Sr-gA
cd Sr-gA
cp .env.example .env
go run ./cmd/srga
```

The server listens on port 4017 and seeds three demo projects on first start. Generate your first sitemap with the `shop-a` key:

```bash
curl -X POST http://localhost:4017/sitemap/generate \
  -H "X-API-Key: shop-a-srga-key-2026" \
  -H "Content-Type: application/json" \
  -d '{"urls": [{"loc": "https://example.com/", "changefreq": "daily", "priority": 1.0}]}'
```

You get XML back plus `X-Sr-gA-URL-Count`, `X-Sr-gA-Cache`, and `X-Sr-gA-Duration-Ms` response headers. Repeat the call and the cache header flips from `miss` to `memory`.

## Run it with Docker

Build once, run anywhere with no toolchain on the host:

```bash
docker compose up -d
curl http://localhost:4017/health
```

The compose file mounts the `srga_data` volume at `/data`, so the SQLite database survives container restarts. To stop and keep your data, run `docker compose down`. To rebuild after code changes, run `docker compose up -d --build`.

## Configuration

Copy `.env.example` to `.env` and edit values. The `.env` file stays out of git by design; commit changes to `.env.example` instead. Every variable has a working default, so you can also run with an empty environment.

| Variable | Default | Purpose |
| :--- | :--- | :--- |
| `PORT` | `4017` | HTTP listen port |
| `ENV` | `production` | Free-form environment label for logs |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error` (JSON via `log/slog`) |
| `DATABASE_PATH` | `./data/srga.db` | SQLite file; parent directories are created on start |
| `MEMORY_CACHE_TTL_SECONDS` | `3600` | In-memory cache entry lifetime |
| `SQLITE_CACHE_TTL_SECONDS` | `86400` | Persistent cache entry lifetime |
| `MEMORY_CACHE_MAX_ITEMS` | `500` | In-memory cache capacity before oldest entries are evicted |
| `MAX_URLS_PER_REQUEST` | `50000` | Largest sitemap request accepted |
| `MAX_URLS_FOR_VALIDATION` | `500` | Largest validation batch accepted |
| `MAX_BATCH_VALIDATION_CONCURRENCY` | `50` | Default goroutine pool size for validation |
| `DEFAULT_VALIDATION_TIMEOUT_SECONDS` | `10` | Per-request HTTP timeout for validation and discovery |
| `VALIDATION_USER_AGENT` | `Sr-gA-Validator/1.0 (+https://github.com/Normious/Sr-gA)` | User-Agent sent when checking remote URLs |
| `VALIDATION_MAX_REDIRECTS` | `5` | Redirect limit before a validation fails |
| `TEMP_DIR` | `./data/tmp` | Scratch directory, created on start |

Demo API keys (`shop-a-srga-key-2026`, `demo-srga-key-2026`, `test-srga-key-2026`) exist for local development only. Create real projects by inserting rows into the `projects` table and rotate the keys before exposing the service.

## Project structure

The layout follows standard Go conventions: `cmd` holds the entrypoint, `internal` holds code that stays inside this module:

```text
srga/
├── cmd/srga/main.go            # wiring, routes, graceful shutdown
├── internal/
│   ├── config/config.go        # env loading and slog setup
│   ├── db/db.go                # SQLite open, migrate, seed
│   ├── db/queries.go           # projects, cache, history, stats
│   ├── cache/memory.go         # in-memory LRU with TTL
│   ├── auth/auth.go            # X-API-Key middleware
│   ├── sitemap/model.go        # XML and JSON types
│   ├── sitemap/generator.go    # urlset, index, gzip
│   ├── sitemap/robots.go       # robots.txt builder
│   ├── sitemap/validator.go    # concurrent HEAD/GET checker
│   ├── sitemap/discover.go     # remote sitemap importer
│   ├── handlers/               # one file per endpoint group
│   └── middleware/middleware.go # recovery, CORS, request logging
├── migrations/0001_init.sql    # full schema, also baked into the image
├── docs/                       # architecture, API reference, ADRs
└── data/                       # local SQLite + temp files (git-ignored)
```

## Testing

Unit tests cover sitemap generation, robots output, and gzip round-trips:

```bash
go test ./...
go vet ./...
```

Endpoint coverage lives in `docs/API.md`, and every example there was executed against a local server and the Docker image before release. To re-run the manual sweep, start the server and work through the API doc top to bottom with the `shop-a` key.

## Further reading

- `docs/API.md`: full endpoint reference with request and response examples
- `docs/ARCHITECTURE.md`: system diagram, request flows, schema, and cache design
- `docs/adr/`: architecture decision records explaining why the stack looks this way
