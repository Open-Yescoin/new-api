> 2026-07-01 历史分析；当前精确源码与图谱基线见 [CURRENT-BASELINE.md](CURRENT-BASELINE.md)。以下事实须按当前源码复核。

<!-- refreshed: 2026-07-01 -->
# Architecture

**Analysis Date:** 2026-07-01

## System Overview

```text
┌─────────────────────────────────────────────────────────────┐
│                    HTTP + Frontend Entry                     │
├──────────────────┬──────────────────┬───────────────────────┤
│   Gin server     │   API routers    │   React console        │
│   `main.go`      │   `router/`      │   `web/src/App.jsx`    │
└────────┬─────────┴────────┬─────────┴──────────┬────────────┘
         │                  │                     │
         ▼                  ▼                     ▼
┌─────────────────────────────────────────────────────────────┐
│             Controller, Middleware, Relay Layer              │
│ `controller/`, `middleware/`, `relay/`, `relay/channel/`     │
└────────┬──────────────────┬─────────────────────┬───────────┘
         │                  │                     │
         ▼                  ▼                     ▼
┌─────────────────────────────────────────────────────────────┐
│                Services, Settings, Models                    │
│ `service/`, `setting/`, `model/`, `dto/`, `common/`          │
└────────┬────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│          SQLite / MySQL / PostgreSQL + Redis/cache           │
│ `model/main.go`, `common/redis.go`, `model/*_cache.go`       │
└─────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| Process bootstrap | Initialize DB/cache/options/background jobs, inject task adaptor factory, create Gin server, mount routes, serve embedded frontend | `main.go` |
| Router layer | Group HTTP routes by API/admin/relay/video/dashboard/web concern and apply auth/rate-limit/distribution middleware | `router/main.go`, `router/api-router.go`, `router/relay-router.go`, `router/video-router.go` |
| Controllers | Thin request handlers, response shaping, retry loops, task creation, admin CRUD orchestration | `controller/relay.go`, `controller/channel.go`, `controller/pricing.go`, `controller/model_meta.go`, `controller/vendor_meta.go`, `controller/task.go` |
| Middleware | Token/user auth, model extraction, channel selection, selected-channel context population, request body storage, logging, rate limiting | `middleware/distributor.go`, `middleware/auth.go`, `middleware/body_storage.go`, `middleware/rate-limit.go` |
| Relay core | Convert unified client requests into upstream provider calls, stream or parse responses, calculate usage and billing | `relay/text.go`, `relay/image_handler.go`, `relay/audio_handler.go`, `relay/relay_task.go`, `relay/relay_adaptor.go` |
| Provider adaptors | Provider-specific request URL/header/body conversion, response parsing, model lists, and channel names | `relay/channel/adapter.go`, `relay/channel/openai/`, `relay/channel/volcengine/`, `relay/channel/task/doubao/` |
| Service layer | Billing, task polling, channel affinity, notifications, sensitive word checks, HTTP client/proxy creation | `service/billing.go`, `service/task_polling.go`, `service/channel.go`, `service/channel_affinity.go`, `service/http_client.go` |
| Model layer | GORM structs, migrations, DB queries, channel/ability/task/user caches, pricing projection | `model/main.go`, `model/channel.go`, `model/ability.go`, `model/task.go`, `model/pricing.go` |
| Settings | Typed accessors around option-backed runtime configuration for ratios, model behavior, operation controls, performance | `setting/ratio_setting/`, `setting/model_setting/`, `setting/operation_setting/`, `setting/system_setting/` |
| React console | Admin and user UI for channels, models/vendors, pricing, tasks, settings, tokens, logs | `web/src/App.jsx`, `web/src/pages/Channel/index.jsx`, `web/src/pages/Model/index.jsx`, `web/src/pages/Pricing/index.jsx`, `web/src/pages/Setting/index.jsx` |

## Pattern Overview

**Overall:** Layered gateway with provider adaptor dispatch.

**Key Characteristics:**
- Keep HTTP routing in `router/`, request orchestration in `controller/`, business rules in `service/`, DB persistence in `model/`, and provider protocol differences in `relay/channel/`.
- Use Gin context keys from `constant/context_key.go` to move authenticated user, token, channel, model, and billing context through middleware, controller, and relay code.
- Use `model.Channel` plus `model.Ability` as the routing source of truth: channels store provider configuration, abilities store enabled `(group, model, channel_id)` routes.
- Use `relay/channel/adapter.go` interfaces for provider-specific code. Normal request providers implement `channel.Adaptor`; async video/task providers implement `channel.TaskAdaptor`, optionally `channel.OpenAIVideoConverter`.
- Preserve upstream compatibility by adding provider changes at existing switch points and constants instead of changing public request/response DTOs.

## Layers

**Bootstrap:**
- Purpose: Initialize process-wide resources and start HTTP serving.
- Location: `main.go`
- Contains: `InitResources()`, cache sync goroutines, option sync, task polling injection, channel model update tasks, Gin middleware setup.
- Depends on: `common/`, `model/`, `controller/`, `service/`, `router/`, `relay/`.
- Used by: The compiled server binary.

**Routing:**
- Purpose: Bind paths to controllers and attach middleware stacks.
- Location: `router/`
- Contains: API routes under `/api`, OpenAI/Claude/Gemini relay routes under `/v1` and `/v1beta`, video routes, dashboard/web routes.
- Depends on: `controller/`, `middleware/`, `relay/`, `types/`.
- Used by: `router.SetRouter()` from `main.go`.

**Controller:**
- Purpose: Validate route-level context, run retry loops, translate service/model results into HTTP responses.
- Location: `controller/`
- Contains: Relay entry points, channel admin CRUD, pricing/model/vendor admin endpoints, payment callbacks, user/token/task/log APIs.
- Depends on: `service/`, `model/`, `relay/`, `setting/`, `dto/`, `types/`.
- Used by: `router/`.

**Middleware:**
- Purpose: Authenticate, rate-limit, decompress/cache request bodies, select channels, set context keys, and log metrics.
- Location: `middleware/`
- Contains: `Distribute()`, `SetupContextForSelectedChannel()`, token/user/admin auth, request decompression, body cleanup.
- Depends on: `model/`, `service/`, `setting/ratio_setting/`, `common/`, `constant/`.
- Used by: `router/relay-router.go`, `router/video-router.go`, `router/api-router.go`.

**Relay:**
- Purpose: Normalize client requests, dispatch to provider adaptors, handle upstream responses, usage accounting, and task submission/fetch.
- Location: `relay/`
- Contains: `TextHelper`, `ImageHelper`, `AudioHelper`, `EmbeddingHelper`, `ResponsesHelper`, `RelayTaskSubmit`, `RelayTaskFetch`, adaptor registry.
- Depends on: `relay/channel/`, `relay/common/`, `relay/helper/`, `service/`, `dto/`, `model/`.
- Used by: `controller/relay.go`.

**Provider Adaptors:**
- Purpose: Isolate upstream provider protocol differences.
- Location: `relay/channel/`
- Contains: One directory per provider (`openai`, `claude`, `gemini`, `volcengine`, etc.) plus async task providers under `relay/channel/task/`.
- Depends on: `dto/`, `relay/common/`, `service/` for HTTP client/error helpers, provider SDKs where needed.
- Used by: `relay.GetAdaptor()` and `relay.GetTaskAdaptor()` in `relay/relay_adaptor.go`.

**Models and Persistence:**
- Purpose: Store and query users, channels, abilities, logs, tasks, pricing metadata, vendors, options, subscriptions, and payment state.
- Location: `model/`
- Contains: GORM models and query helpers. `model/main.go` owns DB initialization/migrations. Cache-aware accessors live in files such as `model/channel_cache.go`.
- Depends on: `gorm.io/gorm`, DB drivers, `common/`, `constant/`, `setting/`.
- Used by: controllers, services, middleware, relay.

**Settings:**
- Purpose: Provide typed access to dynamic options stored in the DB and synced in memory.
- Location: `setting/`
- Contains: ratio, model, operation, performance, console, system, and reasoning settings.
- Depends on: `model/option.go`, `common/`.
- Used by: pricing, billing, distribution, admin settings UI.

**Frontend Console:**
- Purpose: Render admin/user dashboard and call backend APIs.
- Location: `web/src/`
- Contains: route definitions, pages, table components, hooks, helper API wrappers, i18n files.
- Depends on: React 18, Vite, Semi UI, axios, i18next.
- Used by: Users via embedded static build served by `router.SetWebRouter()` or an external `FRONTEND_BASE_URL`.

## Data Flow

### Primary Relay Request Path

1. `main.main()` creates a Gin engine and calls `router.SetRouter(server, buildFS, indexPage)` (`main.go`).
2. `router.SetRelayRouter()` binds `/v1/chat/completions`, `/v1/responses`, `/v1/images/generations`, `/v1/audio/*`, `/v1/embeddings`, `/v1/messages`, `/v1beta/models/*path`, and realtime routes (`router/relay-router.go`).
3. Relay routes apply `middleware.TokenAuth()`, `middleware.ModelRequestRateLimit()`, and `middleware.Distribute()` (`router/relay-router.go`).
4. `middleware.Distribute()` parses the model, validates token model limits, selects a channel via affinity or `service.CacheGetRandomSatisfiedChannel()`, then calls `SetupContextForSelectedChannel()` (`middleware/distributor.go`).
5. `SetupContextForSelectedChannel()` writes channel id/type/key/base URL/model mapping/status-code mapping/settings into Gin context (`middleware/distributor.go`).
6. `controller.Relay()` validates and stores the request body, generates `relaycommon.RelayInfo`, estimates tokens, computes pricing through `relay/helper.ModelPriceHelper()`, pre-consumes quota, and enters a retry loop (`controller/relay.go`).
7. Inside each retry, `getChannel()` reuses the selected channel or asks `service.CacheGetRandomSatisfiedChannel()` for the next satisfied channel and then refreshes selected-channel context (`controller/relay.go`).
8. `controller.Relay()` dispatches by relay format/mode to `relay.TextHelper()`, `relay.ImageHelper()`, `relay.AudioHelper()`, `relay.EmbeddingHelper()`, `relay.ResponsesHelper()`, `relay.ClaudeHelper()`, or `relay.GeminiHelper()` (`controller/relay.go`).
9. Relay helpers call `relay.GetAdaptor(info.ApiType)`, initialize the provider adaptor, apply model mapping, convert the request, call upstream, parse the response, and charge/refund via `service` helpers (`relay/relay_adaptor.go`, `relay/text.go`, `relay/image_handler.go`).
10. Errors are normalized to `types.NewAPIError`; retryable channel errors can disable channels asynchronously through `processChannelError()` and `service.DisableChannel()` (`controller/relay.go`, `service/channel.go`).

### Channel and Provider Dispatch

1. Channel type constants and default base URLs live in `constant/channel.go`.
2. API type constants live in `constant/api_type.go`.
3. `common.ChannelType2APIType()` maps persisted `model.Channel.Type` values to relay API types (`common/api_type.go`).
4. `relay.GetAdaptor()` maps `constant.APIType...` values to provider adaptor structs such as `openai.Adaptor`, `volcengine.Adaptor`, and `deepseek.Adaptor` (`relay/relay_adaptor.go`).
5. Normal provider adaptors implement `channel.Adaptor` from `relay/channel/adapter.go`.
6. New normal providers must update all of `constant/channel.go`, `constant/api_type.go`, `common/api_type.go`, `relay/relay_adaptor.go`, frontend `web/src/constants/channel.constants.js`, and any channel UI behavior in `web/src/components/table/channels/`.
7. BytePlus ModelArk normal chat/completions traffic uses `ChannelTypeVolcEngine` (`45`) and `APITypeVolcEngine`, dispatched to `relay/channel/volcengine/`.

### Async Task and Video Flow

1. `router.SetVideoRouter()` binds `/v1/video/generations`, `/v1/video/generations/:task_id`, `/v1/videos`, `/v1/videos/:task_id`, `/v1/videos/:video_id/remix`, `/v1/videos/:task_id/content`, Kling routes, and Jimeng routes (`router/video-router.go`).
2. Video submit/fetch routes apply `middleware.TokenAuth()` plus `middleware.Distribute()` except the content proxy, which uses `middleware.TokenOrUserAuth()` (`router/video-router.go`).
3. `middleware.Distribute()` sets `relay_mode` for OpenAI-compatible video, legacy video generation, Remix, Kling, Jimeng, and fetch paths (`middleware/distributor.go`).
4. `controller.RelayTask()` generates `relaycommon.RelayInfo`, resolves remix origin task/channel via `relay.ResolveOriginTask()`, runs a retry loop, and calls `relay.RelayTaskSubmit()` (`controller/relay.go`, `relay/relay_task.go`).
5. `relay.RelayTaskSubmit()` chooses a platform from Gin context or `relay.GetTaskPlatform()`, creates a `channel.TaskAdaptor` via `relay.GetTaskAdaptor()`, validates/parses the task request, applies model mapping, computes per-call pricing, pre-consumes quota, sends upstream, parses the task id, and returns `TaskSubmitResult` (`relay/relay_task.go`).
6. On success, `controller.RelayTask()` settles billing, logs consumption, initializes `model.Task`, stores upstream task id in `task.PrivateData.UpstreamTaskID`, stores billing context, and inserts the task (`controller/relay.go`, `model/task.go`).
7. `main.main()` injects `service.GetTaskAdaptorFunc = relay.GetTaskAdaptor` to avoid a service-to-relay import cycle (`main.go`).
8. `controller.UpdateTaskBulk()` delegates to `service.TaskPollingLoop()` (`controller/task.go`).
9. `service.TaskPollingLoop()` sleeps 15 seconds, sweeps timed-out tasks, groups unfinished tasks by platform/channel, and dispatches video updates through `UpdateVideoTasks()` (`service/task_polling.go`).
10. `updateVideoSingleTask()` calls adaptor `FetchTask()`, parses provider status through `ParseTaskResult()`, updates `model.Task` with CAS via `UpdateWithStatus()`, stores result URLs or proxy URLs, and settles/refunds billing (`service/task_polling.go`).
11. `relay.RelayTaskFetch()` serves persisted task data. OpenAI-compatible `/v1/videos/:task_id` responses use `channel.OpenAIVideoConverter` when implemented by the task adaptor (`relay/relay_task.go`).
12. BytePlus/Doubao video routing maps `ChannelTypeDoubaoVideo` and `ChannelTypeVolcEngine` to `relay/channel/task/doubao.TaskAdaptor` (`relay/relay_adaptor.go`). Maintain ModelArk video behavior there unless a separate provider protocol is introduced.

### Pricing, Ratio, and Metadata Flow

1. Admin model/vendor CRUD is exposed through `/api/models` and `/api/vendors` in `router/api-router.go`, implemented by `controller/model_meta.go` and `controller/vendor_meta.go`, backed by `model/model_meta.go` and `model/vendor_meta.go`.
2. Public pricing is exposed through `/api/pricing` in `router/api-router.go`, implemented by `controller.GetPricing()` (`controller/pricing.go`).
3. `model.GetPricing()` refreshes a one-minute cache from enabled abilities, model metadata, vendor metadata, and ratio settings (`model/pricing.go`).
4. `model.updatePricing()` builds `Pricing` rows from `model.GetAllEnableAbilityWithChannels()`, model meta name rules/endpoints, vendor records, and `setting/ratio_setting` model ratios/prices/cache/audio/image ratios (`model/pricing.go`).
5. Ratio admin controls live under `web/src/components/settings/RatioSetting.jsx` and `web/src/pages/Setting/Ratio/`; the pricing page lives under `web/src/pages/Pricing/index.jsx` and `web/src/components/table/model-pricing/`.
6. Group and model pricing docs for Token123 are in `docs/pricing-group-channel-ratio.md`, `docs/token123-pricing-architecture.md`, and `docs/token123-byteplus-pricing-plan.md`.

**State Management:**
- Backend request state is held in Gin context keys from `constant/context_key.go`.
- Persistent runtime state is stored through GORM models in `model/`.
- Dynamic settings are stored as options and synchronized into `setting/` package globals by `model.SyncOptions()` from `main.go`.
- Channel selection state is derived from `model.Ability`, `model.Channel`, channel cache, token group context, and channel affinity service state.
- Task state is persisted in `model.Task` and updated asynchronously by `service.TaskPollingLoop()`.
- Frontend state uses React context under `web/src/context/`, route-local hooks under `web/src/hooks/`, and API helpers under `web/src/helpers/api.js`.

## Key Abstractions

**Channel:**
- Purpose: Persist one upstream provider configuration, key(s), base URL, model list, groups, mapping, overrides, status mappings, and multi-key status.
- Examples: `model/channel.go`, `constant/channel.go`, `web/src/constants/channel.constants.js`
- Pattern: GORM model plus cache helpers. `model.Channel` is selected by `middleware.Distribute()` and populated into context by `SetupContextForSelectedChannel()`.

**Ability:**
- Purpose: Persist enabled routing entries for `(group, model, channel_id)` with priority, weight, and tag.
- Examples: `model/ability.go`, `model/channel_satisfy.go`, `model/channel_cache.go`
- Pattern: Derived from channel models/groups when channels are created or updated. Channel selection queries abilities, not raw channel model strings.

**RelayInfo:**
- Purpose: Carry request, channel, model, price, billing, retry, and task state across controller/relay/helper/adaptor code.
- Examples: `relay/common/relay_info.go`, `controller/relay.go`, `relay/relay_task.go`
- Pattern: Generated once per request by `relaycommon.GenRelayInfo()` and enriched with channel metadata via `InitChannelMeta()`.

**Provider Adaptor:**
- Purpose: Encapsulate upstream normal API protocol conversion and response handling.
- Examples: `relay/channel/adapter.go`, `relay/channel/openai/adaptor.go`, `relay/channel/volcengine/adaptor.go`
- Pattern: Implement `channel.Adaptor`; register in `relay.GetAdaptor()`.

**Task Adaptor:**
- Purpose: Encapsulate async task/video submission, polling, billing adjustment, and result conversion.
- Examples: `relay/channel/adapter.go`, `relay/channel/task/doubao/adaptor.go`, `relay/channel/task/hailuo/adaptor.go`, `relay/channel/task/gemini/adaptor.go`
- Pattern: Implement `channel.TaskAdaptor`; optionally implement `channel.OpenAIVideoConverter`; register in `relay.GetTaskAdaptor()`.

**Pricing Projection:**
- Purpose: Build frontend-friendly model pricing rows from enabled route abilities, model metadata, vendor metadata, and ratio settings.
- Examples: `model/pricing.go`, `controller/pricing.go`, `web/src/components/table/model-pricing/`
- Pattern: One-minute in-memory cache rebuilt by `model.GetPricing()`.

**Settings Options:**
- Purpose: Expose typed runtime configuration that is stored in the options table.
- Examples: `setting/ratio_setting/`, `setting/model_setting/`, `setting/operation_setting/`, `controller/option.go`
- Pattern: Backend option keys are mutated through `/api/option` and read through package-level getters.

## Entry Points

**Server Process:**
- Location: `main.go`
- Triggers: Running the compiled Go binary or container.
- Responsibilities: Resource initialization, background jobs, route mounting, static frontend serving.

**Unified Relay:**
- Location: `controller/relay.go`
- Triggers: `/v1/*`, `/v1beta/*`, `/pg/chat/completions`, `/mj/*`, `/suno/*`.
- Responsibilities: Request validation, pricing, quota pre-consumption, retry, provider dispatch, error normalization.

**Video/Task API:**
- Location: `router/video-router.go`, `controller/relay.go`, `relay/relay_task.go`
- Triggers: `/v1/video/generations`, `/v1/videos`, `/v1/videos/:task_id`, `/v1/videos/:video_id/remix`, provider-specific video routes.
- Responsibilities: Task submit/fetch, persisted task creation, OpenAI video response conversion, proxy content serving.

**Task Poller:**
- Location: `controller/task.go`, `service/task_polling.go`
- Triggers: Goroutine from `main.go` when `common.IsMasterNode && constant.UpdateTask`.
- Responsibilities: Poll unfinished tasks, update statuses/results, settle or refund task billing.

**Admin API:**
- Location: `router/api-router.go`
- Triggers: `/api/channel`, `/api/models`, `/api/vendors`, `/api/pricing`, `/api/ratio_sync`, `/api/task`, `/api/option`.
- Responsibilities: Channel/model/vendor/pricing/settings/task/log/user management.

**Frontend App:**
- Location: `web/src/index.jsx`, `web/src/App.jsx`
- Triggers: Browser navigation served by embedded `web/dist` or external frontend.
- Responsibilities: Route-based console UI, admin pages, pricing page, settings, model/channel/task tables.

## Architectural Constraints

- **Threading:** Go request handlers run concurrently; background goroutines run cache sync, option sync, quota data updates, channel testing, channel upstream model updates, Codex credential refresh, subscription quota reset, and task polling from `main.go`.
- **Database support:** DB code must remain compatible with SQLite, MySQL, and PostgreSQL. Follow `AGENTS.md` and existing compatibility flags in `model/main.go`.
- **JSON calls:** Business code must use `common.Marshal()`, `common.Unmarshal()`, and related wrappers from `common/json.go` rather than direct `encoding/json` marshal/unmarshal calls.
- **Global state:** Many settings and caches are package globals: `common.*`, `setting/*`, `model/pricing.go` caches, `model/channel_cache.go`, `model/user_cache.go`, and `model/channel.go` polling locks.
- **Circular imports:** `service/task_polling.go` intentionally avoids importing `relay`; `main.go` injects `service.GetTaskAdaptorFunc = relay.GetTaskAdaptor`.
- **Provider constants:** Do not add channel types after `ChannelTypeDummy` in `constant/channel.go`; the comment marks it as the count sentinel.
- **Branding/import path:** Project identifiers in `AGENTS.md` are protected. Do not rename module path `github.com/QuantumNous/new-api` or project branding.
- **Task billing:** Async task billing is pre-consumed on submit, stored in `model.Task.PrivateData.BillingContext`, then settled/refunded on final status. Preserve CAS transitions in `service/task_polling.go`.

## Anti-Patterns

### Putting Provider Logic in Controllers

**What happens:** A controller parses provider-specific payloads or calls provider URLs directly.
**Why it's wrong:** It bypasses the `channel.Adaptor`/`channel.TaskAdaptor` boundary and makes upstream sync harder.
**Do this instead:** Put normal provider protocol code in `relay/channel/<provider>/` and async task/video code in `relay/channel/task/<provider>/`, then register it in `relay/relay_adaptor.go`.

### Selecting Channels Outside Distributor/Retry Helpers

**What happens:** Code reads channels directly and manually sets only part of the Gin context.
**Why it's wrong:** Billing, retries, key rotation, model mapping, affinity, status-code mapping, and multi-key logging depend on the full selected-channel context.
**Do this instead:** Let `middleware.Distribute()` select the first channel and use `getChannel()` plus `middleware.SetupContextForSelectedChannel()` on retry (`middleware/distributor.go`, `controller/relay.go`).

### Editing Frontend Channel Labels Without Backend Constants

**What happens:** A channel label is added in `web/src/constants/channel.constants.js` without a matching backend `constant.ChannelType...`, API type mapping, and adaptor registration.
**Why it's wrong:** Admin UI can create channels that the relay layer cannot dispatch.
**Do this instead:** Update `constant/channel.go`, `constant/api_type.go`, `common/api_type.go`, `relay/relay_adaptor.go`, and `web/src/constants/channel.constants.js` together.

### Bypassing Pricing Metadata

**What happens:** Pricing UI or API hardcodes vendor/model display data outside `model/pricing.go`.
**Why it's wrong:** `/api/pricing` already combines abilities, model metadata, vendor metadata, ratio settings, and endpoint support.
**Do this instead:** Update model metadata through `model/model_meta.go` and vendor metadata through `model/vendor_meta.go`; expose UI changes through `web/src/components/table/models/` and `web/src/components/table/model-pricing/`.

### Direct DB-Specific SQL

**What happens:** Raw SQL assumes one database dialect.
**Why it's wrong:** The app supports SQLite, MySQL, and PostgreSQL.
**Do this instead:** Prefer GORM. If raw SQL is required, follow compatibility helpers in `model/main.go` and rules in `AGENTS.md`.

## Error Handling

**Strategy:** Normalize relay errors into OpenAI/Claude/Gemini/task-compatible responses while preserving retry and channel-disable semantics.

**Patterns:**
- Normal relay errors use `types.NewAPIError` and are serialized in `controller.Relay()` based on `types.RelayFormat`.
- Provider HTTP errors pass through helpers such as `service.RelayErrorHandler()` and `service.ResetStatusCode()` (`relay/image_handler.go`, `service/relay_error_handler.go`).
- Task errors use `dto.TaskError` plus wrappers in `service` and are serialized by `respondTaskError()` (`controller/relay.go`).
- Retry decisions are centralized in `shouldRetry()` and `shouldRetryTaskRelay()` (`controller/relay.go`).
- Channel auto-disable decisions flow through `processChannelError()` and `service.ShouldDisableChannel()` (`controller/relay.go`, `service/channel.go`).
- Async task final-state writes use CAS through `model.Task.UpdateWithStatus()` to avoid duplicate billing settlement (`service/task_polling.go`, `model/task.go`).

## Cross-Cutting Concerns

**Logging:** Use `logger/` helpers and `common.SysLog/SysError` for system logs. Relay request logs and error logs are recorded through `model/log.go`, `service` billing/log helpers, and controller error handlers.

**Validation:** Gin middleware handles auth/rate limiting and model extraction. Controllers bind DTOs and validate route-level constraints. Provider adaptors validate provider-specific task requests through `ValidateRequestAndSetAction()`.

**Authentication:** Dashboard/user APIs use session/JWT-style middleware (`middleware.UserAuth()`, `middleware.AdminAuth()`, `middleware.RootAuth()`). Relay APIs use token auth (`middleware.TokenAuth()`), read-only token auth for usage/log reads, and `middleware.TokenOrUserAuth()` for video content proxy.

**Billing:** Normal relay calls pre-consume quota in `controller.Relay()` and settle through relay helpers. Task calls pre-consume in `relay.RelayTaskSubmit()`, settle on submit success in `controller.RelayTask()`, and adjust/refund on poll completion in `service/task_polling.go`.

**Caching:** Channel/model/user/option data can use memory and Redis caches. `main.go` initializes channel cache and starts `model.SyncChannelCache()` and `model.SyncOptions()`. Pricing has an internal one-minute cache in `model/pricing.go`.

**Internationalization:** Backend i18n lives in `i18n/`; frontend i18n lives in `web/src/i18n/locales/`. Frontend translation keys are Chinese source strings per `AGENTS.md`.

**BytePlus ModelArk Maintenance:** Treat `constant.ChannelTypeVolcEngine` and `constant.APITypeVolcEngine` as the normal ModelArk route and `relay/channel/volcengine/` as the provider adaptor. Treat VolcEngine/Doubao async video as `relay/channel/task/doubao/` through `relay.GetTaskAdaptor()` for `ChannelTypeVolcEngine` and `ChannelTypeDoubaoVideo`. Keep pricing metadata in `model/model_meta.go`, `model/vendor_meta.go`, `setting/ratio_setting/`, and docs under `docs/token123-*.md`.

---

*Architecture analysis: 2026-07-01*
