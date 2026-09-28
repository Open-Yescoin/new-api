> 2026-07-01 历史分析；当前精确源码与图谱基线见 [CURRENT-BASELINE.md](CURRENT-BASELINE.md)。以下事实须按当前源码复核。

# External Integrations

**Analysis Date:** 2026-07-01

## APIs & External Services

**AI Relay Providers:**
- Provider integrations are implemented as relay adaptors under `relay/channel/<provider>/` and selected by `relay.GetAdaptor` in `relay/relay_adaptor.go`.
- Channel type IDs and default base URLs live in `constant/channel.go`; adding or maintaining providers starts there, then continues through `relay/channel/*`, `web/src` channel UI, and DB rows in `channels`, `abilities`, `models`, and `vendors`.
- Supported channel families include OpenAI, Azure OpenAI, Anthropic/Claude, Gemini, Vertex AI, AWS Bedrock, OpenRouter, DeepSeek, Volcengine/BytePlus Ark, Ali DashScope, Baidu Qianfan, Tencent Hunyuan, Xunfei, Zhipu, Moonshot, Cohere, Jina, Cloudflare, SiliconFlow, Mistral, Perplexity, xAI, Coze, Dify, Ollama, Xinference, Replicate, Submodel, MiniMax, Jimeng, Kling, Vidu, Suno, Sora, and Codex.
- Request surface supports OpenAI-compatible `/v1/*`, Anthropic `/v1/messages`, Gemini `/v1beta/*`, realtime websocket `/v1/realtime`, image/audio/rerank/responses routes, Midjourney `/mj/*`, Suno `/suno/*`, and video routes under `/v1/video/*` and `/v1/videos/*`. See `router/relay-router.go` and `router/video-router.go`.
- Auth to upstream providers is channel-row API key based. The selected channel key is placed in relay context by distribution middleware and read into `relay/common.RelayInfo` via `InitChannelMeta` in `relay/common/relay_info.go`.

**Token123 / BytePlus ModelArk:**
- Text/chat/image/audio/rerank ModelArk traffic uses `ChannelTypeVolcEngine = 45`, default base URL `https://ark.cn-beijing.volces.com`, and adaptor package `relay/channel/volcengine/`. Request URLs are built in `relay/channel/volcengine/adaptor.go`.
- Doubao/Seedance async video traffic uses `ChannelTypeDoubaoVideo = 54`, default base URL `https://ark.cn-beijing.volces.com`, and task adaptor `relay/channel/task/doubao/`. Do not place video generation on text-only type `45` unless the code path is verified for that model class.
- The Volcengine adaptor handles OpenAI-compatible chat, Claude-format routing for special coding plans, embeddings, images, rerank, responses, and audio speech. Special plan base URL overrides are in `constant/channel.go` under `ChannelSpecialBases`.
- Doubao video submits to `/api/v3/contents/generations/tasks`, fetches `/api/v3/contents/generations/tasks/{task_id}`, maps upstream statuses in `ParseTaskResult`, and converts completed tasks to OpenAI video response shape in `ConvertToOpenAIVideo`.
- Video task billing is asynchronous: `relay/relay_task.go` estimates `OtherRatios`, pre-consumes quota, saves public/upstream task IDs, and `service/task_polling.go` later fetches task status. `service/task_billing.go` refunds failures and recalculates completed tasks when upstream usage tokens or adaptor adjustments are available.
- BytePlus video-input discount ratios are currently hardcoded in `relay/channel/task/doubao/constants.go` for specific Seedance 2.0 models. Keep these in sync with official pricing and DB rollout SQL.
- Token123 skill policy for ModelArk maintenance:
  - Use target group `byteplus`.
  - Keep `byteplus` `GroupRatio` at `1`.
  - Display currency is CNY.
  - Do not change production `USDExchangeRate` by default.
  - For text token models, configure ratio billing with `ModelRatio`, `CompletionRatio`, and `CacheRatio`; do not configure fixed `ModelPrice`.
  - Cache-hit pricing is independent and must use official cache-hit pricing, not a derived input-price shortcut.
  - Public models also need `vendors` and `models` rows with a BytePlus `vendor_id`, non-empty description, enabled ability rows, and correct `channels.models` / `channels.model_mapping`.
  - Never expose or copy upstream API keys into Markdown.

**Model Metadata Sync:**
- Upstream model metadata sync endpoints are implemented in `controller/model_sync.go` and exposed under `/api/models/sync_upstream/*` in `router/api-router.go`.
- Default sync base is `SYNC_UPSTREAM_BASE`, falling back to `https://basellm.github.io/llm-metadata` in `controller/model_sync.go`.
- Admin model/vendor metadata endpoints are in `controller/model_meta.go`, `controller/vendor_meta.go`, and routes `/api/models/*`, `/api/vendors/*`.
- Channel upstream model update detection and application are implemented in `controller/channel_upstream_update.go` and exposed under `/api/channel/upstream_updates/*`.

**Model Deployment / IO.NET:**
- Deployment management uses `pkg/ionet/` with default API bases `https://api.io.solutions/v1/io-cloud/caas` and `https://api.io.solutions/enterprise/v1/io-cloud/caas` in `pkg/ionet/client.go`.
- Admin routes under `/api/deployments/*` are registered in `router/api-router.go` and call controller code in `controller/deployment.go`.
- API auth uses `X-API-KEY` in `pkg/ionet/client.go`.

**Notifications and Webhooks:**
- Outgoing generic webhook notifications use `service.SendWebhookNotify` in `service/webhook.go`.
- Outgoing webhook requests can be sent directly with SSRF validation or through a configured worker using `system_setting.WorkerUrl` / `WorkerValidKey` from `setting/system_setting/system_setting_old.go`.
- Webhook signing uses HMAC-SHA256 in `service/webhook.go` with `X-Webhook-Signature` and optional bearer auth when a secret is configured.
- Fetch and webhook SSRF controls are configured through `setting/system_setting/fetch_setting.go` and enforced by `common/ssrf_protection.go` plus `service/http_client.go`.

**Analytics and Observability Vendors:**
- Umami injection is controlled by `UMAMI_WEBSITE_ID` and `UMAMI_SCRIPT_URL` in `main.go`.
- Google Analytics injection is controlled by `GOOGLE_ANALYTICS_ID` in `main.go`.
- Prometheus metrics are exposed by `router/metrics-router.go` and middleware in `middleware/metrics.go`.
- Pyroscope profiling is optional via `PYROSCOPE_URL` and related env vars in `common/pyro.go`.
- pprof is optional via `ENABLE_PPROF=true` in `main.go`, serving on port `8005`.

## Data Storage

**Databases:**
- Primary SQL database is selected in `model/main.go` from `SQL_DSN`.
  - Empty `SQL_DSN` or local mode uses SQLite through `github.com/glebarez/sqlite`; database path defaults to `one-api.db?_busy_timeout=30000` from `common/database.go` and can be changed with `SQLITE_PATH`.
  - `postgres://` or `postgresql://` DSNs use PostgreSQL through `gorm.io/driver/postgres`.
  - Other non-empty DSNs use MySQL through `gorm.io/driver/mysql`; `parseTime=true` is appended when missing.
- Optional log database uses `LOG_SQL_DSN`; when unset, `LOG_DB` points to the primary DB in `model/main.go`.
- GORM automigration covers channels, tokens, users, passkeys, options, redemptions, abilities, logs, Midjourney, topups, commission records, quota data, tasks, models, vendors, prefill groups, setup, 2FA, checkins, subscriptions, custom OAuth providers, and OAuth bindings in `model/main.go`.
- DB connection pool env vars are `SQL_MAX_IDLE_CONNS`, `SQL_MAX_OPEN_CONNS`, and `SQL_MAX_LIFETIME`.

**File Storage:**
- Local filesystem storage is used for SQLite, logs, embedded frontend assets, and disk cache.
- Disk cache behavior is implemented in `common/disk_cache.go`, `common/disk_cache_config.go`, and `service/file_service.go`.
- Uploaded/downloaded media is loaded from URLs or base64 through `service/file_service.go`, with request-scoped cleanup via `middleware/body_cleanup.go`.
- External object storage SDKs are not detected as primary storage clients. Provider channels may fetch media URLs from upstream services.

**Caching:**
- Redis is optional and enabled only when `REDIS_CONN_STRING` is set; initialization is in `common/redis.go`.
- Redis client package: `github.com/go-redis/redis/v8`.
- Memory cache is enabled by `MEMORY_CACHE_ENABLED=true`; Redis also forces memory cache on in `main.go` for compatibility.
- Hybrid cache helper is in `pkg/cachex/hybrid_cache.go`; channel/token/user caches live in `model/channel_cache.go`, `model/token_cache.go`, and `model/user_cache.go`.
- Sync frequency is controlled by `SYNC_FREQUENCY`; channel cache and options sync loops run from `main.go`.

## Authentication & Identity

**Local Auth:**
- Password login/register uses user routes in `router/api-router.go`, controller logic in `controller/user.go`, and session cookies from `gin-contrib/sessions` in `main.go`.
- Session signing uses `SESSION_SECRET`; `CRYPTO_SECRET` falls back to `SESSION_SECRET` in `common/init.go`.
- API token auth and read-only token auth are implemented in `middleware/auth.go` and token management in `controller/token.go`, `model/token.go`.

**OAuth Providers:**
- OAuth registry and provider interface live in `oauth/registry.go` and `oauth/provider.go`.
- Built-in providers:
  - GitHub: `oauth/github.go`, enabled by DB options `GitHubOAuthEnabled`, `GitHubClientId`, `GitHubClientSecret`.
  - Discord: `oauth/discord.go`, settings registered in `setting/system_setting/discord.go`.
  - OIDC: `oauth/oidc.go`, settings registered in `setting/system_setting/oidc.go`.
  - Linux DO: `oauth/linuxdo.go`, supports env endpoint overrides `LINUX_DO_TOKEN_ENDPOINT` and `LINUX_DO_USER_ENDPOINT`.
  - Telegram and WeChat are non-standard auth routes in `router/api-router.go`, implemented by `controller/telegram.go` and `controller/wechat.go`.
- Custom OAuth providers are DB-backed. `oauth.LoadCustomProviders` loads `model.CustomOAuthProvider` rows at startup from `model/custom_oauth_provider.go`; admin CRUD routes are `/api/custom-oauth-provider/*` in `router/api-router.go`.
- Custom OAuth supports token/userinfo endpoints, auth style, and access policy evaluation in `oauth/generic.go`.

**MFA and Anti-Abuse:**
- Passkeys/WebAuthn use `service/passkey/`, `controller/passkey.go`, and `model/passkey.go`.
- TOTP/2FA uses `controller/twofa.go`, `model/twofa.go`, and `github.com/pquerna/otp`.
- Cloudflare Turnstile is enforced by `middleware/turnstile-check.go`; options include `TurnstileSiteKey`, `TurnstileSecretKey`, and `TurnstileCheckEnabled` in `model/option.go`.
- Rate limits are configured by env vars in `common/init.go` and enforced by middleware such as `middleware/rate-limit.go`, `middleware/model-rate-limit.go`, and `middleware/email-verification-rate-limit.go`.

## Payments and Billing Vendors

**EPay:**
- SDK: `github.com/Calcium-Ion/go-epay/epay`.
- Top-up implementation: `controller/topup.go`; subscription EPay implementation: `controller/subscription_payment_epay.go`.
- Routes: `/api/user/epay/notify`, `/api/subscription/epay/notify`, `/api/subscription/epay/return` in `router/api-router.go`.
- Settings: `PayAddress`, `EpayId`, `EpayKey`, `PayMethods` in `model/option.go` and `setting/operation_setting/payment_setting_old.go`.

**Stripe:**
- SDK: `github.com/stripe/stripe-go/v81`.
- Checkout and webhook implementation: `controller/topup_stripe.go`; subscription payment implementation: `controller/subscription_payment_stripe.go`.
- Routes: `/api/user/stripe/pay`, `/api/user/stripe/amount`, `/api/stripe/webhook`, `/api/subscription/stripe/pay`.
- Settings: `StripeApiSecret`, `StripeWebhookSecret`, `StripePriceId`, `StripeUnitPrice`, `StripeMinTopUp`, `StripePromotionCodesEnabled` in `setting/payment_stripe.go` and `model/option.go`.
- Webhook verification uses Stripe signature header and `webhook.ConstructEventWithOptions` in `controller/topup_stripe.go`.

**Creem:**
- HTTP client integration in `controller/topup_creem.go`; no dedicated Go SDK detected.
- Checkout endpoint target is selected in `genCreemLink` in `controller/topup_creem.go`.
- Routes: `/api/user/creem/pay`, `/api/creem/webhook`, `/api/subscription/creem/pay`.
- Settings: `CreemApiKey`, `CreemProducts`, `CreemTestMode`, `CreemWebhookSecret` in `setting/payment_creem.go` and `model/option.go`.
- Webhook verification uses `creem-signature` HMAC in `controller/topup_creem.go`.

**Waffo:**
- SDK: `github.com/waffo-com/waffo-go`.
- Implementation: `controller/topup_waffo.go`; optional Pancake variant is present in `controller/topup_waffo_pancake.go` and settings in `setting/payment_waffo_pancake.go`.
- Routes: `/api/user/waffo/amount`, `/api/user/waffo/pay`, `/api/waffo/webhook`.
- Settings: `WaffoEnabled`, API keys, private/public certificates, sandbox variants, merchant ID, notify/return URLs, currency, unit price, minimum top-up, and pay methods in `setting/payment_waffo.go` and `model/option.go`.
- Webhook verification and signed response use Waffo SDK methods in `controller/topup_waffo.go`.

## Monitoring & Observability

**Error Tracking:**
- External error tracking service not detected.
- Application logging uses internal logger package `logger/logger.go`, system logging helpers in `common/sys_log.go`, and log persistence through `model/log.go`.
- Optional error log behavior is controlled by `ERROR_LOG_ENABLED` in `common/init.go`.

**Logs:**
- Request logging is installed through `middleware.SetUpLogger(server)` in `main.go`.
- Consumption and billing logs are recorded in `model/log.go`, `service/log_info_generate.go`, `service/task_billing.go`, and relay handlers.
- Admin log APIs are registered under `/api/log/*` in `router/api-router.go`.
- Log DB may be split from primary DB with `LOG_SQL_DSN`.

**Metrics:**
- Prometheus middleware is installed in `main.go` and routes are registered by `router/metrics-router.go`.
- Runtime performance APIs are under `/api/performance/*` in `router/api-router.go`.
- Uptime Kuma status integration is exposed at `/api/uptime/status` and implemented by `controller/uptime_kuma.go`; configuration is part of operation settings.

## CI/CD & Deployment

**Hosting:**
- Docker image target is `calciumion/new-api`; workflows also support GHCR for alpha builds.
- `Dockerfile` builds a static-ish Go binary and runs it from `/data` on port `3000`.
- `docker-compose.yml` includes `new-api`, `redis`, and `postgres` services. Do not quote or commit compose env values that contain credentials.
- Desktop packaging uses Electron under `electron/`, with app binaries embedding built backend and frontend resources.

**CI Pipeline:**
- `.github/workflows/docker-image-alpha.yml` builds and signs alpha multi-arch Docker images for Docker Hub and GHCR.
- `.github/workflows/docker-image-arm64.yml` builds native amd64/arm64 tag images and multi-arch Docker Hub manifests.
- `.github/workflows/release.yml` builds Linux/macOS/Windows backend release binaries and uploads GitHub Releases.
- `.github/workflows/electron-build.yml` builds Windows Electron artifacts.
- `.github/workflows/pr-check.yml` runs `peakoss/anti-slop` on PRs.
- `.github/workflows/sync-to-gitee.yml` syncs GitHub releases to Gitee.

## Environment Configuration

**Required env vars:**
- Minimum production: `SESSION_SECRET` plus durable `SQL_DSN` for PostgreSQL/MySQL or durable `SQLITE_PATH`/volume for SQLite.
- Multi-node/cache: `REDIS_CONN_STRING`, consistent `SESSION_SECRET`, appropriate `NODE_TYPE`, and shared DB.
- Relay tuning: `RELAY_TIMEOUT`, `RELAY_MAX_IDLE_CONNS`, `RELAY_MAX_IDLE_CONNS_PER_HOST`, `SYNC_FREQUENCY`.
- Async task operation: `UPDATE_TASK`, `TASK_QUERY_LIMIT`, `TASK_TIMEOUT_MINUTES`, and Redis/cache settings for high-volume polling.
- Security/rate-limit: `CRITICAL_RATE_LIMIT_*`, `GLOBAL_API_RATE_LIMIT_*`, `GLOBAL_WEB_RATE_LIMIT_*`, `SEARCH_RATE_LIMIT_*`, `TRUSTED_REDIRECT_DOMAINS`.
- Optional integrations: provider channel keys in `channels.key`, OAuth/payment secrets in DB `options`, `PYROSCOPE_*`, analytics env vars, `FRONTEND_BASE_URL`.

**Secrets location:**
- Runtime env: `.env` is loaded locally by `main.go`; `.env.example` exists but is not read or quoted here.
- DB-backed secrets: `options` table stores OAuth, SMTP, payment, webhook, worker, and Turnstile settings via `model/option.go`.
- Channel secrets: `channels.key` stores upstream API keys; multi-key state is in `model.ChannelInfo` in `model/channel.go`.
- Never place provider API keys, payment secrets, private keys, certificates, or SQL passwords in `.planning/` Markdown.

## Webhooks & Callbacks

**Incoming:**
- Stripe: `POST /api/stripe/webhook` -> `controller.StripeWebhook` in `controller/topup_stripe.go`.
- Creem: `POST /api/creem/webhook` -> `controller.CreemWebhook` in `controller/topup_creem.go`.
- Waffo: `POST /api/waffo/webhook` -> `controller.WaffoWebhook` in `controller/topup_waffo.go`.
- EPay top-up: `GET|POST /api/user/epay/notify` -> `controller.EpayNotify` in `controller/topup.go`.
- EPay subscription: `GET|POST /api/subscription/epay/notify` and `/api/subscription/epay/return` -> `controller/subscription_payment_epay.go`.
- Midjourney callbacks are represented by Midjourney relay/task controllers and options such as `MjNotifyEnabled` in `model/option.go`; routes live under `/mj/*` in `router/relay-router.go`.
- Jimeng and Kling official-compatible video task routes convert incoming vendor-shaped requests via `middleware/jimeng_adapter.go` and `middleware/kling_adapter.go`.

**Outgoing:**
- Generic webhook notifications: `service.SendWebhookNotify` posts to configured webhook URLs, with optional HMAC signature and optional worker proxy.
- Payment redirect/callback URLs are built from `service.GetCallbackAddress`, `system_setting.ServerAddress`, and vendor-specific option overrides in payment controllers.
- OAuth redirects use `system_setting.ServerAddress` for Discord/OIDC/custom providers and request host logic for Linux DO in `oauth/linuxdo.go`.
- Upstream AI calls use `service.GetHttpClientWithProxy`, per-channel proxy settings from `dto.ChannelSettings`, SSRF-aware redirect checks, and provider-specific request builders under `relay/channel/*`.

## New-API Channel / Provider Pattern

**Add or maintain a synchronous provider:**
- Define or verify channel type/default base URL in `constant/channel.go`.
- Implement `channel.Adaptor` in `relay/channel/<provider>/` using the interface from `relay/channel/adapter.go`.
- Register adaptor selection in `relay.GetAdaptor` in `relay/relay_adaptor.go`.
- Confirm API-type mapping in `common/api_type.go` and endpoint/relay mode behavior in `relay/relay_adaptor.go`, `relay/helper/model_mapped.go`, and `relay/common/relay_info.go`.
- Add channel rows with `type`, `key`, `base_url`, `models`, `model_mapping`, group, priority/weight, and settings through `/api/channel/*` or SQL.
- Add/refresh `abilities` through `model.Channel.UpdateAbilities` in `model/ability.go`.
- Add marketplace `vendors` and `models` rows through `/api/vendors/*`, `/api/models/*`, or migration SQL.

**Add or maintain an asynchronous task/video provider:**
- Implement `channel.TaskAdaptor` from `relay/channel/adapter.go`.
- Register it in `relay.GetTaskAdaptor` in `relay/relay_adaptor.go`.
- Ensure submit path in `relay/relay_task.go` validates request, maps model names, estimates billing, pre-consumes quota, and stores task IDs.
- Ensure polling path in `service/task_polling.go` can fetch, parse, mark terminal states, refund failures, and recalculate actual quota.
- For OpenAI-compatible video fetches, implement `OpenAIVideoConverter` like `relay/channel/task/doubao/adaptor.go`.
- Maintain task pricing in `setting/ratio_setting/model_ratio.go`, `setting/ratio_setting/group_ratio.go`, and provider-specific ratio code such as `relay/channel/task/doubao/constants.go`.

**ModelArk Maintenance Checklist:**
- Use `ChannelTypeVolcEngine=45` for text/token/image/rerank ModelArk channels and `ChannelTypeDoubaoVideo=54` for Doubao/Seedance video task channels.
- Verify upstream endpoint IDs, model aliases, official input/output/cache/video prices, and region before DB changes.
- Configure `channels.models`, `channels.model_mapping`, `abilities.enabled`, `models.vendor_id`, `models.status`, `models.deleted_at`, and `vendors` together.
- Use ratio billing for text token models. Keep fixed `ModelPrice` for image/video/per-call models only when the code path expects per-call billing.
- For video aliases, publish conservative high-price aliases unless request validation/enforcement guarantees resolution/audio/draft constraints.
- After DB rollout, restart `new-api` when required, then verify `/api/status`, vendor/model filtering, channel tests, and task billing logs.

---

*Integration audit: 2026-07-01*
