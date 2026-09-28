> 2026-07-01 历史分析；当前精确源码与图谱基线见 [CURRENT-BASELINE.md](CURRENT-BASELINE.md)。以下事实须按当前源码复核。

# Technology Stack

**Analysis Date:** 2026-07-01

## Languages

**Primary:**
- Go 1.25.1 - backend API gateway, relay adapters, persistence, billing, auth, task polling, and embedded web server. Declared in `go.mod`.
- JavaScript / JSX - React dashboard in `web/src`, Vite config in `web/vite.config.js`, Electron shell in `electron/`.

**Secondary:**
- YAML - i18n locale files in `i18n/locales/`, GitHub Actions in `.github/workflows/`.
- Lua - Redis limiter script support under `common/limiter/lua/`.
- Shell-compatible Dockerfile syntax - container build in `Dockerfile`.

## Runtime

**Environment:**
- Backend executable: `main.go` starts Gin on `PORT` or `--port`, defaults to `3000` in `common/init.go`.
- Go module target: `go 1.25.1` in `go.mod`.
- Container builder: `golang:1.26.1-alpine` with `CGO_ENABLED=0`, `GOEXPERIMENT=greenteagc`, and final image `debian:bookworm-slim` in `Dockerfile`.
- Frontend dev runtime: Node-compatible Vite app in `web/`; CI uses Bun in `.github/workflows/release.yml` and `.github/workflows/electron-build.yml`.
- Electron desktop runtime: Electron `39.8.5` in `electron/package.json`.

**Package Manager:**
- Go modules: `go.mod` and `go.sum`.
- Frontend: pnpm lockfile `web/pnpm-lock.yaml` with lockfile version `9.0`; CI uses `bun install` and `bun run build`.
- Electron: npm lockfile `electron/package-lock.json` with lockfile version `3`.
- Lockfiles: present for Go, web frontend, and Electron.

## Frameworks

**Core:**
- Gin `github.com/gin-gonic/gin` `v1.9.1` - HTTP routing, middleware, API dashboard, relay endpoints. Entry points: `main.go`, `router/main.go`, `router/api-router.go`, `router/relay-router.go`, `router/video-router.go`.
- GORM `gorm.io/gorm` `v1.25.2` - ORM and migrations. Main initialization: `model/main.go`.
- React `^18.2.0` - web dashboard under `web/src`, built by Vite in `web/vite.config.js`.
- Vite `5.1.0` - frontend dev server/build tool. Config: `web/vite.config.js`.
- Electron `39.8.5` with `electron-builder` `^26.7.0` - desktop packaging under `electron/`.

**Testing:**
- Go standard `testing` plus `github.com/stretchr/testify` `v1.11.1` - backend unit tests such as `service/task_billing_test.go`, `model/task_cas_test.go`, `controller/channel_upstream_update_test.go`.
- Frontend test runner not detected; `web/package.json` contains lint/build scripts but no test script.
- Electron test runner not detected.

**Build/Dev:**
- Backend local build: `go build -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$(cat VERSION)'" -o new-api` mirrors `Dockerfile`.
- Backend local run: `go run . --port 3000` or built binary `./new-api --port 3000`; flags are defined in `common/init.go`.
- Frontend dev: `cd web && pnpm install && pnpm run dev`; Vite proxies `/api`, `/mj`, and `/pg` to `http://localhost:3000` in `web/vite.config.js`.
- Frontend build: `cd web && pnpm run build` or CI-equivalent `cd web && bun install && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$VERSION bun run build`.
- Docker build: `docker build -t new-api .`; final container exposes `3000` and runs `/new-api` from `/data`.
- Docker compose: `docker-compose.yml` defines `new-api`, `redis`, and `postgres` services; inspect env values without committing secrets.
- Electron dev/build: `cd electron && npm run dev-app`, `npm run build:win`, `npm run build:mac`, or `npm run build:linux`.

## Key Dependencies

**Critical:**
- `github.com/gin-gonic/gin` `v1.9.1` - request routing, middleware, sessions, and relay endpoint dispatch.
- `gorm.io/gorm` `v1.25.2` plus `gorm.io/driver/mysql`, `gorm.io/driver/postgres`, `github.com/glebarez/sqlite` - SQLite/MySQL/PostgreSQL support in `model/main.go`.
- `github.com/go-redis/redis/v8` `v8.11.5` - optional Redis cache/rate-limit/task support initialized in `common/redis.go`.
- `github.com/gorilla/websocket` `v1.5.0` - realtime relay support referenced by `relay/common/relay_info.go`.
- `github.com/tiktoken-go/tokenizer` `v0.6.2` - token counting initialized from `service/tokenizer.go` and `service/token_counter.go`.
- `github.com/aws/aws-sdk-go-v2/service/bedrockruntime` `v1.50.4` - AWS Bedrock channel in `relay/channel/aws/`.
- `github.com/stripe/stripe-go/v81` `v81.4.0` - Stripe checkout and webhook handling in `controller/topup_stripe.go`.
- `github.com/waffo-com/waffo-go` `v1.3.1` - Waffo payment orders and webhook signing in `controller/topup_waffo.go`.
- `github.com/Calcium-Ion/go-epay` `v0.0.4` - EPay payments in `controller/topup.go` and `controller/subscription_payment_epay.go`.
- `github.com/go-webauthn/webauthn` `v0.14.0` and `github.com/pquerna/otp` `v1.5.0` - passkeys and TOTP in `service/passkey/`, `controller/passkey.go`, and `controller/twofa.go`.

**Frontend:**
- `@douyinfe/semi-ui` `2.69.1` and `@douyinfe/semi-icons` `2.63.1` - dashboard UI components.
- `axios` `1.15.0` - web API client.
- `react-router-dom` `^6.3.0` and `history` `^5.3.0` - dashboard routing.
- `i18next`, `react-i18next`, `i18next-browser-languagedetector` - frontend localization.
- `@visactor/vchart` and `@visactor/react-vchart` - dashboard charts.
- `react-turnstile` and `react-telegram-login` - Cloudflare Turnstile and Telegram login UI integrations.
- `@lobehub/icons`, `react-icons`, `lucide-react` - model/vendor icons and UI icons.

**Infrastructure:**
- `github.com/prometheus/client_golang` `v1.22.0` - metrics router in `router/metrics-router.go` and middleware in `middleware/metrics.go`.
- `github.com/grafana/pyroscope-go` `v1.2.7` - optional profiling in `common/pyro.go`.
- `github.com/gin-contrib/sessions`, `github.com/gin-contrib/cors`, `github.com/gin-contrib/gzip`, `github.com/gin-contrib/static` - session/cors/compression/static web serving.
- `github.com/joho/godotenv` `v1.5.1` - `.env` loading in `main.go`; do not commit env files or values.
- `github.com/samber/lo` and `github.com/samber/hot` - collection helpers and in-memory cache utilities.

## Configuration

**Environment:**
- `.env.example` is present but not read by this map because env-like files are treated as secret-bearing. Runtime loads `.env` if present via `godotenv.Load(".env")` in `main.go`.
- Process env and flags are parsed in `common/init.go`; use env vars for deployment settings and DB options, not hardcoded source changes.
- Critical runtime env vars used by code:
  - `PORT`, `VERSION`, `GIN_MODE`, `DEBUG`
  - `SESSION_SECRET`, `CRYPTO_SECRET`, `SQLITE_PATH`
  - `SQL_DSN`, `LOG_SQL_DSN`, `SQL_MAX_IDLE_CONNS`, `SQL_MAX_OPEN_CONNS`, `SQL_MAX_LIFETIME`
  - `REDIS_CONN_STRING`, `REDIS_POOL_SIZE`, `SYNC_FREQUENCY`, `MEMORY_CACHE_ENABLED`
  - `NODE_TYPE`, `NODE_NAME`, `FRONTEND_BASE_URL`
  - `RELAY_TIMEOUT`, `RELAY_MAX_IDLE_CONNS`, `RELAY_MAX_IDLE_CONNS_PER_HOST`, `TLS_INSECURE_SKIP_VERIFY`
  - `GLOBAL_API_RATE_LIMIT_*`, `GLOBAL_WEB_RATE_LIMIT_*`, `CRITICAL_RATE_LIMIT_*`, `SEARCH_RATE_LIMIT_*`
  - `STREAMING_TIMEOUT`, `MAX_FILE_DOWNLOAD_MB`, `STREAM_SCANNER_MAX_BUFFER_MB`, `MAX_REQUEST_BODY_MB`
  - `UPDATE_TASK`, `TASK_QUERY_LIMIT`, `TASK_TIMEOUT_MINUTES`, `TASK_PRICE_PATCH`
  - `CHANNEL_UPDATE_FREQUENCY`, `CHANNEL_TEST_FREQUENCY`, `CHANNEL_UPSTREAM_MODEL_UPDATE_TASK_ENABLED`
  - `AZURE_DEFAULT_API_VERSION`, `GEMINI_SAFETY_SETTING`, `COHERE_SAFETY_SETTING`
  - `PYROSCOPE_URL`, `PYROSCOPE_APP_NAME`, `PYROSCOPE_BASIC_AUTH_USER`, `PYROSCOPE_BASIC_AUTH_PASSWORD`
  - `UMAMI_WEBSITE_ID`, `UMAMI_SCRIPT_URL`, `GOOGLE_ANALYTICS_ID`
  - `SYNC_UPSTREAM_BASE`, `SYNC_HTTP_TIMEOUT_SECONDS`, `SYNC_HTTP_RETRY`, `SYNC_HTTP_MAX_MB`
  - `TRUSTED_REDIRECT_DOMAINS`

**Database-backed Options:**
- Application settings live in the `options` table via `model.Option` in `model/option.go`.
- Registered structured settings use `setting/config/config.go`; modules register under keys such as `discord`, `oidc`, `fetch_setting`, `payment_setting`, `group_ratio_setting`.
- Channel/provider config is stored in `channels` and `abilities` through `model/channel.go` and `model/ability.go`.
- Marketplace metadata is stored in `vendors` and `models` through `model/vendor_meta.go`, `model/model_meta.go`, and controller routes in `controller/vendor_meta.go` and `controller/model_meta.go`.

**Build:**
- Backend build config: `go.mod`, `go.sum`, `Dockerfile`, `VERSION`.
- Frontend build config: `web/package.json`, `web/pnpm-lock.yaml`, `web/vite.config.js`.
- Electron build config: `electron/package.json`, `electron/package-lock.json`, `electron/main.js`, `electron/preload.js`.
- CI/CD config: `.github/workflows/docker-image-alpha.yml`, `.github/workflows/docker-image-arm64.yml`, `.github/workflows/release.yml`, `.github/workflows/electron-build.yml`, `.github/workflows/pr-check.yml`, `.github/workflows/sync-to-gitee.yml`.

## Platform Requirements

**Development:**
- Go `>=1.25.1` for local backend builds. CI uses `actions/setup-go` with `>=1.25.1`; Docker uses Go `1.26.1`.
- Node-compatible runtime for `web/`; repository lockfile is pnpm, while CI uses Bun. Use one package manager per working tree to avoid lockfile churn.
- Docker and Docker Compose for containerized local stack. `docker-compose.yml` includes `new-api`, `redis`, and `postgres`.
- PostgreSQL, MySQL, or SQLite. SQLite is default when `SQL_DSN` is unset or local mode is selected in `model/main.go`; production README recommends MySQL or PostgreSQL.
- Redis is optional but recommended for multi-node/cache-heavy deployments and required by some polling/cache workflows; initialized only when `REDIS_CONN_STRING` is set.

**Production:**
- Single binary serves API, relay, metrics, and embedded `web/dist` unless `FRONTEND_BASE_URL` redirects web routes on non-master nodes. Router logic is in `router/main.go`.
- Container image published as `calciumion/new-api` by `.github/workflows/docker-image-alpha.yml` and `.github/workflows/docker-image-arm64.yml`.
- Multi-node deployments require consistent `SESSION_SECRET`, shared SQL database, and Redis-backed cache/sync settings.
- Master/slave behavior is controlled with `NODE_TYPE`; database migrations, setup checks, channel cache/task updates, and frontend serving depend on master-node behavior in `main.go` and `router/main.go`.
- Token123/new-api ModelArk maintenance must preserve ratio-billing conventions: target group `byteplus`, `GroupRatio` `1`, no fixed `ModelPrice` for text token models, and no API keys in Markdown.

---

*Stack analysis: 2026-07-01*
