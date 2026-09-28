> 2026-07-01 历史分析；当前精确源码与图谱基线见 [CURRENT-BASELINE.md](CURRENT-BASELINE.md)。以下事实须按当前源码复核。

# Codebase Structure

**Analysis Date:** 2026-07-01

## Directory Layout

```text
new-api/
├── main.go                 # Process bootstrap, background jobs, Gin server
├── go.mod                  # Go module `github.com/QuantumNous/new-api`
├── AGENTS.md               # Project conventions and protected rules
├── common/                 # Shared utilities, env, JSON wrappers, cache, quota, endpoints
├── constant/               # Channel/API/task/context/endpoint constants
├── controller/             # Gin handlers and request orchestration
├── dto/                    # Request/response DTO structs
├── i18n/                   # Backend translations
├── logger/                 # Logging helpers
├── metrics/                # Prometheus metrics
├── middleware/             # Auth, distributor, logging, rate limits, body storage
├── model/                  # GORM models, migrations, DB/cache query helpers
├── oauth/                  # OAuth provider implementations
├── pkg/                    # Internal packages such as cachex and ionet
├── relay/                  # Unified relay helpers, task relay, provider adaptor registry
│   ├── channel/            # Provider protocol adaptors
│   ├── common/             # RelayInfo, overrides, task request helpers
│   ├── common_handler/     # Shared relay handlers
│   ├── constant/           # Relay mode constants/path mapping
│   ├── helper/             # Pricing/model mapping/request helpers
│   └── reasonmap/          # Provider finish reason mapping
├── router/                 # Route registration by route family
├── service/                # Business services: billing, tasks, affinity, payment, HTTP clients
├── setting/                # Option-backed typed settings
├── types/                  # Shared type definitions and error wrappers
├── web/                    # React/Vite/Semi UI frontend
│   ├── src/
│   ├── public/
│   └── dist/
├── docs/                   # Project and Token123 operational docs
├── deploy/                 # Deployment support files
├── bin/                    # Migration scripts and small utilities
└── .planning/codebase/     # Generated codebase map documents
```

## Directory Purposes

**`common/`:**
- Purpose: Cross-cutting utilities used by nearly every backend layer.
- Contains: JSON wrappers, env parsing, quota helpers, endpoint defaults/types, request body storage, Redis setup, URL validation, crypto, paging, system logs.
- Key files: `common/json.go`, `common/api_type.go`, `common/endpoint_defaults.go`, `common/body_storage.go`, `common/redis.go`, `common/database.go`, `common/quota.go`.

**`constant/`:**
- Purpose: Stable enum-like values shared across backend packages.
- Contains: Channel types, API types, context keys, task platforms/actions, endpoint types, cache keys, setup constants.
- Key files: `constant/channel.go`, `constant/api_type.go`, `constant/context_key.go`, `constant/task.go`, `constant/endpoint_type.go`.

**`router/`:**
- Purpose: Register HTTP routes and middleware stacks.
- Contains: API/admin routes, relay routes, video routes, dashboard/web routes, metrics route.
- Key files: `router/main.go`, `router/api-router.go`, `router/relay-router.go`, `router/video-router.go`, `router/web-router.go`.

**`middleware/`:**
- Purpose: Apply request-level cross-cutting behavior before controllers.
- Contains: Auth, distributor/channel selection, body storage, decompression, metrics, CORS, i18n, rate limits, secure verification.
- Key files: `middleware/distributor.go`, `middleware/auth.go`, `middleware/body_storage.go`, `middleware/rate-limit.go`, `middleware/logger.go`.

**`controller/`:**
- Purpose: Gin handlers that orchestrate service/model/relay calls and shape HTTP responses.
- Contains: Relay entry, channel admin, model/vendor admin, pricing, tasks, logs, users, tokens, payments, settings.
- Key files: `controller/relay.go`, `controller/channel.go`, `controller/pricing.go`, `controller/model_meta.go`, `controller/vendor_meta.go`, `controller/task.go`, `controller/option.go`, `controller/ratio_sync.go`.

**`relay/`:**
- Purpose: Provider-agnostic relay logic and provider adaptor registry.
- Contains: Text/image/audio/embedding/responses helpers, task submit/fetch, Midjourney handling, adaptor factory, relay common data.
- Key files: `relay/relay_adaptor.go`, `relay/text.go`, `relay/image_handler.go`, `relay/audio_handler.go`, `relay/relay_task.go`, `relay/common/relay_info.go`, `relay/helper/model_price.go`.

**`relay/channel/`:**
- Purpose: Provider-specific normal request adaptors.
- Contains: `adapter.go` interfaces plus provider folders such as `openai/`, `claude/`, `gemini/`, `volcengine/`, `deepseek/`, `aws/`, `cohere/`.
- Key files: `relay/channel/adapter.go`, `relay/channel/openai/adaptor.go`, `relay/channel/volcengine/adaptor.go`, `relay/channel/openai/relay-openai.go`.

**`relay/channel/task/`:**
- Purpose: Provider-specific async task/video adaptors.
- Contains: Task adaptors for Ali, Doubao/VolcEngine, Gemini, Hailuo/MiniMax, Jimeng, Kling, Sora/OpenAI, Suno, Vertex, Vidu.
- Key files: `relay/channel/task/doubao/adaptor.go`, `relay/channel/task/doubao/constants.go`, `relay/channel/task/hailuo/adaptor.go`, `relay/channel/task/gemini/adaptor.go`, `relay/channel/task/sora/adaptor.go`, `relay/channel/task/taskcommon/helpers.go`.

**`model/`:**
- Purpose: Persistence, migrations, cache-aware DB access, and pricing projection.
- Contains: GORM structs and methods for channels, abilities, users, tokens, tasks, logs, models, vendors, subscriptions, options.
- Key files: `model/main.go`, `model/channel.go`, `model/ability.go`, `model/channel_cache.go`, `model/channel_satisfy.go`, `model/task.go`, `model/pricing.go`, `model/model_meta.go`, `model/vendor_meta.go`, `model/option.go`.

**`service/`:**
- Purpose: Business logic that is not route-specific and not provider protocol-specific.
- Contains: Billing, task polling, channel affinity, sensitive word checks, HTTP clients, payment helpers, subscriptions, notifications.
- Key files: `service/billing.go`, `service/task_polling.go`, `service/task_billing.go`, `service/channel.go`, `service/channel_affinity.go`, `service/http_client.go`, `service/post_consume.go`.

**`setting/`:**
- Purpose: Typed access to option-backed runtime settings.
- Contains: Ratio, model, operation, performance, console, system, reasoning settings.
- Key files: `setting/ratio_setting/`, `setting/model_setting/`, `setting/operation_setting/`, `setting/performance_setting/`, `setting/system_setting/`.

**`web/src/`:**
- Purpose: React frontend source.
- Contains: App routes, pages, table components, settings components, hooks, API helpers, i18n files, styles.
- Key files: `web/src/App.jsx`, `web/src/index.jsx`, `web/src/helpers/api.js`, `web/src/constants/channel.constants.js`, `web/src/pages/Channel/index.jsx`, `web/src/pages/Model/index.jsx`, `web/src/pages/Pricing/index.jsx`, `web/src/pages/Setting/index.jsx`, `web/src/pages/Task/index.jsx`.

**`docs/`:**
- Purpose: Project docs, channel docs, OpenAPI docs, and Token123 operational notes.
- Contains: Pricing architecture notes, BytePlus pricing plan, change-log schema notes, channel docs, installation docs.
- Key files: `docs/token123-pricing-architecture.md`, `docs/token123-byteplus-pricing-plan.md`, `docs/pricing-group-channel-ratio.md`, `docs/token123-change-log-schema-move-20260623.md`.

## Key File Locations

**Entry Points:**
- `main.go`: Backend process entry, resource initialization, background goroutines, route mounting.
- `web/src/index.jsx`: React root render.
- `web/src/App.jsx`: Frontend route definitions for console, pricing, auth, settings, and task pages.

**Routing:**
- `router/main.go`: Top-level router composition.
- `router/api-router.go`: `/api` admin/user/public endpoints.
- `router/relay-router.go`: OpenAI/Claude/Gemini/Midjourney/Suno relay endpoints.
- `router/video-router.go`: OpenAI-compatible video/task endpoints and video content proxy.
- `router/web-router.go`: Embedded frontend/static route handling.

**Configuration:**
- `go.mod`: Go module and backend dependencies.
- `web/package.json`: Frontend dependencies and scripts.
- `.env.example`: Example environment configuration; do not read real `.env` files.
- `Dockerfile`, `docker-compose.yml`, `new-api.service`: Deployment/runtime examples. Treat inline secrets in compose files as sensitive if present.
- `AGENTS.md`: Mandatory project conventions, including JSON wrapper usage and DB compatibility.

**Core Relay Logic:**
- `controller/relay.go`: Primary normal relay and task relay controllers.
- `relay/relay_adaptor.go`: Normal and task adaptor factory switches.
- `relay/channel/adapter.go`: `Adaptor`, `TaskAdaptor`, and `OpenAIVideoConverter` interfaces.
- `relay/common/relay_info.go`: Relay context object.
- `relay/helper/model_price.go`: Relay pricing helper entry points.
- `middleware/distributor.go`: Model extraction, channel selection, and selected-channel context setup.

**Channel and Provider Definitions:**
- `constant/channel.go`: Channel type ids, names, default base URLs.
- `constant/api_type.go`: Relay API type ids.
- `common/api_type.go`: Channel type to API type mapping.
- `web/src/constants/channel.constants.js`: Frontend channel type list and model-fetchable type set.
- `model/channel.go`: Channel model, keys, base URL/settings accessors, multi-key behavior.
- `model/ability.go`: Ability model and channel ability generation.
- `model/channel_satisfy.go`: Channel satisfaction checks for group/model/channel routing.

**BytePlus ModelArk Locations:**
- `constant/channel.go`: `ChannelTypeVolcEngine = 45`, default `https://ark.cn-beijing.volces.com`.
- `constant/api_type.go`: `APITypeVolcEngine`.
- `common/api_type.go`: `ChannelTypeVolcEngine` maps to `APITypeVolcEngine`.
- `relay/relay_adaptor.go`: `APITypeVolcEngine` returns `volcengine.Adaptor`; task `ChannelTypeVolcEngine` returns `taskdoubao.TaskAdaptor`.
- `relay/channel/volcengine/`: Normal VolcEngine/ModelArk chat and TTS adaptor files.
- `relay/channel/task/doubao/`: Doubao/VolcEngine async video adaptor files.
- `web/src/constants/channel.constants.js`: Admin UI label for channel type `45`.
- `docs/token123-byteplus-pricing-plan.md`: Token123 BytePlus pricing plan notes.

**Pricing and Model Metadata:**
- `controller/pricing.go`: `/api/pricing` response composition.
- `model/pricing.go`: Pricing projection and cache.
- `model/pricing_default.go`: Default vendor inference rules/icons.
- `model/model_meta.go`: Model metadata persistence.
- `model/vendor_meta.go`: Vendor metadata persistence.
- `setting/ratio_setting/`: Model/group ratio and price settings.
- `web/src/components/settings/RatioSetting.jsx`: Settings tab shell for ratio controls.
- `web/src/pages/Setting/Ratio/`: Admin ratio/pricing editors.
- `web/src/components/table/model-pricing/`: Public pricing page tables/cards/filters.
- `web/src/components/table/models/`: Admin model/vendor metadata tables and modals.

**Task and Video:**
- `router/video-router.go`: Video/task endpoint registration.
- `relay/relay_task.go`: Task submit/fetch, remix origin resolution, OpenAI video conversion fetch path.
- `service/task_polling.go`: Async polling loop, status updates, result URLs, billing settle/refund.
- `model/task.go`: Task model, task private data, billing context, status helpers.
- `controller/video_proxy.go`, `controller/video_proxy_gemini.go`: Video content proxy and realtime fetch support.
- `web/src/pages/Task/index.jsx`, `web/src/components/table/task-logs/`, `web/src/hooks/task-logs/useTaskLogsData.js`: Task log UI.

**Testing:**
- Backend tests are co-located with packages using `*_test.go`, such as `controller/channel_upstream_update_test.go`, `relay/channel/deepseek/adaptor_test.go`, `service/task_billing_test.go`, `model/task_cas_test.go`.
- Frontend test structure is not prominent in `web/src/`; frontend scripts expose lint/build checks in `web/package.json`.

## Naming Conventions

**Go Files:**
- Use package-oriented lowercase file names: `relay/relay_task.go`, `model/vendor_meta.go`, `controller/channel_upstream_update.go`.
- Tests use `*_test.go` next to implementation files: `relay/channel/api_request_test.go`, `service/task_billing_test.go`.
- Provider directories use lowercase names matching provider packages: `relay/channel/volcengine/`, `relay/channel/deepseek/`, `relay/channel/task/doubao/`.

**Go Types and Functions:**
- Exported API uses PascalCase: `Channel`, `Ability`, `RelayTaskSubmit`, `GetPricing`.
- Unexported helpers use camelCase: `getChannel`, `shouldRetryTaskRelay`, `updateVideoSingleTask`.
- Provider adaptors consistently use `Adaptor` for normal providers and `TaskAdaptor` for async task providers.

**Constants:**
- Channel type constants use `ChannelTypeProvider`: `constant.ChannelTypeVolcEngine`, `constant.ChannelTypeDoubaoVideo`.
- API type constants use `APITypeProvider`: `constant.APITypeVolcEngine`.
- Context keys live in `constant/context_key.go`.
- Task platforms/actions live in `constant/task.go`.

**Frontend Files:**
- React pages use directory `index.jsx`: `web/src/pages/Channel/index.jsx`, `web/src/pages/Pricing/index.jsx`.
- Components use PascalCase filenames: `EditChannelModal.jsx`, `PricingTable.jsx`, `ModelPricingCombined.jsx`.
- Hooks use `use*.js` or `use*.jsx`: `web/src/hooks/channels/useChannelsData.jsx`, `web/src/hooks/model-pricing/useModelPricingData.jsx`.
- Constants use lowercase domain names: `web/src/constants/channel.constants.js`.

**Directories:**
- Admin table domains live under `web/src/components/table/<domain>/`.
- Settings subpages live under `web/src/pages/Setting/<area>/`.
- Shared UI components live under `web/src/components/common/`.

## Where to Add New Code

**New Normal Provider/Channel:**
- Backend constants: `constant/channel.go`, `constant/api_type.go`.
- Channel-to-API mapping: `common/api_type.go`.
- Adaptor implementation: `relay/channel/<provider>/adaptor.go`, plus provider-specific `constants.go`, `dto.go`, `relay-*.go`, or `text.go` as needed.
- Adaptor registration: `relay/relay_adaptor.go` in `GetAdaptor()`.
- Channel UI option: `web/src/constants/channel.constants.js`.
- Channel admin behavior: `web/src/components/table/channels/modals/EditChannelModal.jsx`, `web/src/components/table/channels/ChannelsColumnDefs.jsx`, `web/src/hooks/channels/useChannelsData.jsx`.
- Model fetch support: add the channel type to `MODEL_FETCHABLE_CHANNEL_TYPES` in `web/src/constants/channel.constants.js` only when backend fetch-model support exists.
- Tests: provider unit tests under `relay/channel/<provider>/` and routing/billing tests under `controller/` or `service/` when behavior crosses layers.

**BytePlus ModelArk Normal API Changes:**
- Primary code: `relay/channel/volcengine/`.
- Dispatch constants: `constant/channel.go`, `constant/api_type.go`, `common/api_type.go`, `relay/relay_adaptor.go`.
- Channel UI label/settings: `web/src/constants/channel.constants.js`, `web/src/components/table/channels/`.
- Pricing/model metadata: `model/model_meta.go`, `model/vendor_meta.go`, `model/pricing.go`, `setting/ratio_setting/`, `web/src/components/table/models/`, `web/src/pages/Setting/Ratio/`.
- Operational docs: `docs/token123-byteplus-pricing-plan.md`, `docs/token123-pricing-architecture.md`.

**New Async Video/Task Provider:**
- Task adaptor: `relay/channel/task/<provider>/adaptor.go`.
- Shared task helper reuse: `relay/channel/task/taskcommon/helpers.go`.
- Task adaptor registration: `relay.GetTaskAdaptor()` in `relay/relay_adaptor.go`.
- Task platform/action constants: `constant/task.go` if a string platform/action is needed.
- Channel type and UI option: `constant/channel.go`, `web/src/constants/channel.constants.js`.
- Video routes: `router/video-router.go` only when adding a new public route family.
- Middleware model extraction: `middleware/distributor.go` when new routes need custom model/action parsing.
- Task fetch/response conversion: implement `channel.OpenAIVideoConverter` when `/v1/videos/:task_id` must return OpenAI video format.
- Tests: `service/task_billing_test.go`, provider adaptor tests under `relay/channel/task/<provider>/`, and controller tests for route/retry behavior.

**BytePlus/Doubao Video Changes:**
- Primary code: `relay/channel/task/doubao/adaptor.go` and `relay/channel/task/doubao/constants.go`.
- Dispatch: `relay.GetTaskAdaptor()` maps both `constant.ChannelTypeDoubaoVideo` and `constant.ChannelTypeVolcEngine` to `taskdoubao.TaskAdaptor`.
- Polling behavior: `service/task_polling.go`.
- Task persistence fields: `model/task.go`.
- Video content proxy behavior: `controller/video_proxy.go`, `controller/video_proxy_gemini.go`.
- Frontend task logs: `web/src/pages/Task/index.jsx`, `web/src/components/table/task-logs/`.

**Pricing, Ratio, or Vendor Changes:**
- Backend pricing projection: `model/pricing.go`.
- Default vendor mapping/icons: `model/pricing_default.go`.
- Vendor CRUD: `model/vendor_meta.go`, `controller/vendor_meta.go`, route group in `router/api-router.go`.
- Model metadata CRUD: `model/model_meta.go`, `controller/model_meta.go`, route group in `router/api-router.go`.
- Ratio settings: `setting/ratio_setting/`, `controller/pricing.go`, `controller/option.go`.
- Admin ratio UI: `web/src/components/settings/RatioSetting.jsx`, `web/src/pages/Setting/Ratio/`.
- Public pricing UI: `web/src/pages/Pricing/index.jsx`, `web/src/components/table/model-pricing/`, `web/src/hooks/model-pricing/`.
- Admin model/vendor UI: `web/src/pages/Model/index.jsx`, `web/src/components/table/models/`, `web/src/hooks/models/useModelsData.jsx`.

**Channel Admin UI Changes:**
- Page shell: `web/src/pages/Channel/index.jsx`.
- Table: `web/src/components/table/channels/ChannelsTable.jsx`, `ChannelsColumnDefs.jsx`, `ChannelsActions.jsx`, `ChannelsFilters.jsx`, `ChannelsTabs.jsx`.
- Edit/create modal: `web/src/components/table/channels/modals/EditChannelModal.jsx`.
- Model selection: `web/src/components/table/channels/modals/ModelSelectModal.jsx`, `SingleModelSelectModal.jsx`.
- Upstream model update UI: `web/src/components/table/channels/modals/ChannelUpstreamUpdateModal.jsx`, `web/src/hooks/channels/useChannelUpstreamUpdates.jsx`.
- Data hook: `web/src/hooks/channels/useChannelsData.jsx`.
- Backend API: `controller/channel.go`, channel route group in `router/api-router.go`.

**Admin API Endpoint:**
- Route: Add to the appropriate group in `router/api-router.go`.
- Handler: Add to `controller/<domain>.go`.
- Business logic: Put reusable logic in `service/<domain>.go`.
- Persistence: Put GORM structs/queries in `model/<domain>.go`.
- Frontend call: Add or reuse helpers in `web/src/helpers/api.js` and domain hook under `web/src/hooks/`.

**Database Schema/Model:**
- GORM model: `model/<domain>.go`.
- Migration registration: `model/main.go`.
- Keep SQLite/MySQL/PostgreSQL compatibility. Use `commonGroupCol`, `commonKeyCol`, `commonTrueVal`, and `commonFalseVal` patterns from `model/main.go` when raw SQL is unavoidable.
- Token123 operational backup/change-log tables: use PostgreSQL schema `token123_change_log` and names beginning with `token123_` per `AGENTS.md`.

**Frontend Page:**
- Route: `web/src/App.jsx`.
- Page shell: `web/src/pages/<PageName>/index.jsx`.
- Components: `web/src/components/<domain>/` or `web/src/components/table/<domain>/`.
- Hooks: `web/src/hooks/<domain>/`.
- Constants: `web/src/constants/<domain>.constants.js`.
- Translations: `web/src/i18n/locales/*.json`; use Chinese source strings as keys.

**Utilities:**
- Backend shared helpers: `common/` only for broadly reused utilities.
- Relay-only helpers: `relay/helper/` or `relay/common/`.
- Service/business helpers: `service/`.
- Frontend shared helpers: `web/src/helpers/`.

## Special Directories

**`.codegraph/`:**
- Purpose: Local CodeGraph index used for code intelligence.
- Generated: Yes.
- Committed: No; local index files such as `.codegraph/codegraph.db` are workspace artifacts.

**`.planning/codebase/`:**
- Purpose: Generated architecture, structure, stack, quality, and concerns reference documents for GSD workflows.
- Generated: Yes.
- Committed: Intended as planning artifacts.

**`web/dist/`:**
- Purpose: Built frontend assets embedded/served by the backend.
- Generated: Yes.
- Committed: Present in the repository snapshot.

**`docs/`:**
- Purpose: Human-maintained project, channel, OpenAPI, pricing, and Token123 operational documentation.
- Generated: Mixed; treat markdown docs as source artifacts unless clearly generated by a doc tool.
- Committed: Yes.

**`bin/`:**
- Purpose: SQL migrations and utility scripts.
- Generated: No.
- Committed: Yes.

**`deploy/`:**
- Purpose: Deployment support configuration such as Token123 docs Nginx config.
- Generated: No.
- Committed: Yes.

**`electron/`:**
- Purpose: Electron packaging/runtime support.
- Generated: Mixed.
- Committed: Yes.

**`.env*`:**
- Purpose: Environment configuration.
- Generated: No.
- Committed: `.env.example` only. Do not read or copy real `.env` contents.

---

*Structure analysis: 2026-07-01*
