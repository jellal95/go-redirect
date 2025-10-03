# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

Project: go-redirect (Go + Fiber)

Quick commands
- Build (local)
  - go build -o bin/go-redirect .
- Run (local dev)
  - PORT=8080 go run .
  - Defaults: LOG_PATH=./logs (created automatically), templates in ./views, config at ./config/config.yaml, GeoLite2-*.mmdb in repo root.
- Tests
  - All tests: go test ./...
  - Single test: go test ./handlers -run TestRedirectHandler -v
- Format and static checks (no repo-specific linter configured)
  - go fmt ./...
  - go vet ./...
- Docker
  - docker build -t go-redirect .
  - docker run -e PORT=8080 -p 8080:8080 go-redirect
- Fly.io deploy
  - CI: pushes to main trigger .github/workflows/fly-deploy.yml
  - Manual: flyctl deploy --remote-only

High-level architecture
- Entry point: main.go
  - Loads application configuration from config/config.yaml (products, ad network postback URLs, bot filter settings).
  - Initializes GeoIP databases:
    - City DB via geo.InitGeoDB("GeoLite2-City.mmdb") for analytics enrichment.
    - Country DB passed to middleware.NewBotFilter("GeoLite2-Country.mmdb") for allowlist filtering.
  - Sets up Fiber v2 with HTML template engine (github.com/gofiber/template/html/v2) rendering views/ .
  - Global middleware: RequestID, plus conditional bot filter and a lightweight request logger on selected routes.
  - Listens on PORT (default 8080).

- Routing (handlers/*)
  - Public, unfiltered utilities: /health, /ready, /logs (aggregated JSONL analytics), /dashboard (templated), /sse (server-sent events), /postbacks (in-memory view), /article, /main, /agoda-verification-page.
  - Postback ingress (logging only): GET /postback — accepts partner callbacks, logs, and forwards to configured networks (Propeller, Galaksion, Popcash, ClickAdilla) by building query strings (handlers/postback.go).
  - Core traffic flows (protected by conditional bot filter + request logger):
    - GET / — RedirectHandler picks a product by percentage (from config) or by explicit product query, enriches with geo/user-agent/headers, builds final affiliate URL via utils.BuildAffiliateURL, and issues a 302.
    - GET /pre-sale — PreSaleHandler selects and previews a product; can directly redirect when redirect=direct, preserving campaign query parameters and tagging from=presale.
  - Bot filter control: POST /toggle-bot-filter and GET /bot-filter-status flip/read global filter state (handlers/bot_filter_toggle.go with middleware/conditional_bot_filter.go).

- Middleware (middleware/*)
  - bot_middleware: Implements IP rate limiting (sliding window), user-agent and referrer blacklists (including regex), geo allowlist (by country code), and optional mobile-only gating. Also supports a fixed bypass key via query/header. Uses GeoLite2 Country DB when configured.
  - requestid: Adds X-Request-ID to each request using crypto/rand fallback.
  - logger: Simple console request logs with timing and referer context.

- Domain models (models/*)
  - models.go: Product, GeoInfo, LogEntry types and constants for route/postback kinds and ad network IDs.
  - config.go: Structs that mirror config/config.yaml (ad networks, bot filter, products).

- Utilities (utils/*)
  - config.go: YAML config loader into models.Config.
  - csv.go: Loader for config/config.csv with Indonesian column headers; computes product weights from Komisi/Komisi hingga when needed (used by MainHandler/PreSale as fallback or data source).
  - helpers.go: BuildAffiliateURL replaces placeholders like {sub_id}, {siteid}, {type_ads} with values from incoming query params, skips unresolved placeholders, and appends remaining campaign params deterministically.
  - logger.go: Structured in-memory + JSONL file logging (WIB timezone). File destination controlled by LOG_PATH; default ./logs in development, /logs in production. Provides LogsSummary() for SSE and appends one JSON line per entry.

- Geo (geo/*)
  - geo.go: Thin wrapper over github.com/oschwald/geoip2-golang for city lookups used in analytics enrichment. Returns "Unknown" fields when DB not present or lookup fails.

Runtime and environment
- Required files at runtime (expected relative to repo root):
  - GeoLite2-City.mmdb, GeoLite2-Country.mmdb
  - config/config.yaml (and optionally config/config.csv)
  - views/*.html
- Environment variables
  - PORT: HTTP server port (default 8080)
  - LOG_PATH: folder for JSONL logs; defaults to ./logs locally. In Fly.io, set to /logs and persisted via [[mounts]] in fly.toml

Deployment
- Fly.io configuration in fly.toml
  - Overrides Docker ARG GO_VERSION to 1.24 during remote build.
  - Mounts a volume at /logs for persisted logs and sets PORT/LOG_PATH.
- Dockerfile
  - Multi-stage build, copies config/, views/, and GeoLite2-*.mmdb into runtime image; entrypoint is the built binary run-app.
- GitHub Actions
  - .github/workflows/fly-deploy.yml deploys on push to main via flyctl deploy --remote-only.

Notes
- Tests currently cover redirect URL construction logic in handlers/redirect_handler_test.go; run them when changing utils/helpers.go or handlers/redirect.go.
- The app logs analytics to JSONL; use GET /logs to retrieve aggregated summaries for dashboards. Ensure LOG_PATH is writable in your environment.
