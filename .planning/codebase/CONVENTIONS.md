> 2026-07-01 历史分析；当前精确源码与图谱基线见 [CURRENT-BASELINE.md](CURRENT-BASELINE.md)。以下事实须按当前源码复核。

# Coding Conventions

**Analysis Date:** 2026-07-01

## Naming Patterns

**Files:**
- Go packages use lowercase directory names and snake_case test/source files for multiword concepts: `relay/helper/stream_scanner.go`, `relay/common/stream_status.go`, `service/task_billing.go`.
- Provider relay packages live under `relay/channel/{provider}/` and expose provider-specific files such as `relay/channel/openai/adaptor.go`, `relay/channel/claude/adaptor.go`, and `relay/channel/aws/adaptor.go`.
- Async task/video providers live under `relay/channel/task/{provider}/` and use `TaskAdaptor` implementations, for example `relay/channel/task/doubao/adaptor.go` and `relay/channel/task/vidu/adaptor.go`.
- Frontend React components use PascalCase `.jsx` files: `web/src/components/table/channels/ChannelsTable.jsx`, `web/src/components/settings/RatioSetting.jsx`.
- Frontend hooks use `useXxx` names and live under feature folders: `web/src/hooks/channels/useChannelsData.jsx`, `web/src/hooks/model-pricing/useModelPricingData.jsx`.
- Frontend constants use `.constants.js` suffixes: `web/src/constants/channel.constants.js`, `web/src/constants/channel-affinity-template.constants.js`.

**Functions:**
- Go exported functions use PascalCase and local helpers use camelCase: `GetAdaptor` in `relay/relay_adaptor.go`, `buildCompletionRatioMetaValue` in `controller/option.go`.
- Controller handlers are exported verbs/nouns that receive `*gin.Context`: `GetOptions` and `UpdateOption` in `controller/option.go`.
- Model mutation helpers use direct action names such as `UpdateOption`, `DecreaseUserQuota`, `UpdateChannelUsedQuota`, and `RecordConsumeLog` across `model/option.go`, `model/user.go`, `model/channel.go`, and `model/log.go`.
- Provider adaptors implement stable method names: `Init`, `GetRequestURL`, `SetupRequestHeader`, `ConvertOpenAIRequest`, `DoRequest`, `DoResponse`, and `GetModelList` in files like `relay/channel/claude/adaptor.go`.
- Task/video adaptors implement task-specific verbs: `ValidateRequestAndSetAction`, `BuildRequestURL`, `BuildRequestHeader`, `BuildRequestBody`, `DoRequest`, `DoResponse`, and `FetchTask` in `relay/channel/task/doubao/adaptor.go`.
- React components use PascalCase functions, while hooks and event handlers use camelCase: `Pricing` in `web/src/pages/Pricing/index.jsx`, `handleGroupClick` in `web/src/hooks/model-pricing/useModelPricingData.jsx`.

**Variables:**
- Go package-level mutable state is common for settings and caches; protect shared maps with locks when present: `common.OptionMapRWMutex` in `model/option.go`, `updatePricingLock` and `modelEnableGroupsLock` in `model/pricing.go`.
- Go error variables are usually returned inline rather than declared globally; use `types.ErrorCode...` constants for relay errors from `types/error.go`.
- Frontend state variables use `[value, setValue]` from React hooks: `filterVendor`, `setFilterVendor`, `models`, and `setModels` in `web/src/hooks/model-pricing/useModelPricingData.jsx`.
- Frontend local constants are uppercase for fixed maps and keys, for example `COLUMN_KEYS` in `web/src/hooks/channels/useChannelsData.jsx`.

**Types:**
- Go DTO/model structs use PascalCase type names and JSON/GORM tags: `Task`, `TaskPrivateData`, and `TaskBillingContext` in `model/task.go`.
- Optional scalar request fields that relay upstream JSON must use pointer-style optional wrappers or pointer scalars so explicit zero values survive marshal/unmarshal. Examples are covered by `dto/openai_request_zero_value_test.go` and `dto/gemini_generation_config_test.go`.
- Relay errors use `*types.NewAPIError` from `types/error.go`; task/video errors use `*dto.TaskError` wrapped by `service.TaskErrorWrapper` in `service/error.go`.

## Code Style

**Formatting:**
- Use `gofmt` for all Go files. Imports are grouped as standard library, local module packages, then third-party packages as seen in `relay/channel/task/doubao/adaptor.go`.
- Frontend formatting is Prettier-based. `web/.prettierrc.mjs` extends `@so1ve/prettier-config`, and `web/package.json` also sets single quotes for JavaScript and JSX.
- Frontend source files carry the AGPL/QuantumNous block header enforced by `web/.eslintrc.cjs`. Add it to new `.js` and `.jsx` files under `web/src`.

**Linting:**
- Go lint config is not detected. Use `go test ./...` as the baseline compile/static check for backend changes.
- Frontend lint commands are defined in `web/package.json`: `bun run lint` runs `prettier . --check`, and `bun run eslint` runs ESLint over `**/*.{js,jsx}`.
- ESLint enforces the license header and limits consecutive blank lines through `web/.eslintrc.cjs`.

## Import Organization

**Order:**
1. Go standard library imports such as `fmt`, `io`, `net/http`, `strings`.
2. Local module imports under `github.com/QuantumNous/new-api/...`.
3. Third-party imports such as `github.com/gin-gonic/gin`, `github.com/pkg/errors`, `github.com/stretchr/testify/require`.

**Path Aliases:**
- Backend imports use the module path from `go.mod`: `github.com/QuantumNous/new-api`.
- Frontend Vite alias `@` points to `web/src` in `web/vite.config.js`, but existing files mostly use relative imports such as `../../helpers` and `../../constants`.

## Error Handling

**Patterns:**
- Use `types.NewError`, `types.NewOpenAIError`, or related constructors from `types/error.go` for relay errors that must carry retry, status, masking, or provider-format metadata.
- Use `types.ErrorCode...` constants from `types/error.go` instead of ad hoc string codes in relay/common code.
- Use `service.RelayErrorHandler` in `service/error.go` to parse upstream HTTP error bodies and normalize them into `*types.NewAPIError`.
- Use `service.TaskErrorWrapper` or `service.TaskErrorFromAPIError` in `service/error.go` for async task/video paths.
- Wrap provider conversion errors with context when returning them: `errors.Wrap(err, "convert request payload failed")` in `relay/channel/task/doubao/adaptor.go`.
- Mask sensitive upstream/network errors before exposing them. `types.NewAPIError.MaskSensitiveError` in `types/error.go` and `service.TaskErrorWrapper` in `service/error.go` perform this.
- Gin controllers return JSON objects with `success`, `message`, and optional `data`, as in `controller/option.go`.
- Frontend API errors flow through the Axios instance in `web/src/helpers/api.js`; pass `skipErrorHandler` in request config when a call needs local error handling.

## Logging

**Framework:** Custom `common.SysLog` and `logger` package.

**Patterns:**
- Use `common.SysLog` for startup/configuration and migration events, for example `model/main.go`.
- Use `logger.LogInfo`, `logger.LogWarn`, and `logger.LogError` with a context for runtime service paths, for example `service/task_billing.go`.
- Do not log raw API keys, tokens, or provider secrets. Mask user-visible error text via `common.MaskSensitiveInfo`, `types.NewAPIError.MaskSensitiveError`, or task wrappers.
- Consumption and billing logs are domain records, not process logs. Use `model.RecordConsumeLog` and `model.RecordTaskBillingLog` as in `service/task_billing.go`.

## Comments

**When to Comment:**
- Keep comments for behavior that prevents regressions across providers, billing semantics, or compatibility. Good examples include `TaskBillingContext` comments in `model/task.go`, task/video billing comments in `service/task_billing.go`, and header passthrough behavior in `relay/channel/api_request.go`.
- Comments in English or Chinese both exist. Match nearby code; do not translate or churn existing comments during focused changes.
- Avoid deleting protected project identity or license comments. `AGENTS.md` marks references to `new-api` and `QuantumNous` as protected.

**JSDoc/TSDoc:**
- Not detected as a standard. Frontend code relies on clear component/hook names and inline comments rather than JSDoc.

## Function Design

**Size:** Keep new helpers small and testable around parsing, normalization, conversion, and billing math. Existing focused helpers include `normalizeModelNames` in `controller/channel_upstream_update.go`, `parseStatusCodeMappingValue` in `service/error.go`, and `hasVideoInMetadata` in `relay/channel/task/doubao/adaptor.go`.

**Parameters:** Prefer passing `*gin.Context` and `*relay/common.RelayInfo` through relay paths rather than reconstructing request state. Provider adaptors should mutate `RelayInfo` only for established fields such as `UpstreamModelName`, `FinalRequestRelayFormat`, and runtime header override state.

**Return Values:** Go functions return `(value, error)` for ordinary failures and `*types.NewAPIError`/`*dto.TaskError` for relay/task failures that have API semantics. Avoid panic outside test setup and startup initialization.

## Module Design

**Exports:** Export functions used across layers; keep provider-specific helpers unexported unless tests in the same package need them. Examples: `GetAdaptor` in `relay/relay_adaptor.go` is exported, while `hasVideoInMetadata` in `relay/channel/task/doubao/adaptor.go` is unexported.

**Barrel Files:** Frontend uses `index.jsx` or `index.js` entry files for feature folders such as `web/src/components/table/channels/index.jsx` and `web/src/pages/Pricing/index.jsx`.

## Config/Option Patterns

- Options are initialized with defaults in `model.InitOptionMap` in `model/option.go`, loaded from the `options` table, then mirrored into package-level runtime variables.
- Option updates enter through `controller.UpdateOption` in `controller/option.go`, where request values are decoded with `common.DecodeJson`, converted to strings, validated by option key, and persisted through `model.UpdateOption`.
- `model.UpdateOption` in `model/option.go` uses `DB.FirstOrCreate`, `DB.Save`, then `updateOptionMap` under `common.OptionMapRWMutex`.
- Hierarchical config keys such as `performance_setting.*` are delegated through `handleConfigUpdate` in `model/option.go` and `setting/config/config.go`.
- Pricing options are stringified JSON maps. The important keys are `ModelRatio`, `ModelPrice`, `CompletionRatio`, `CacheRatio`, `CreateCacheRatio`, `ImageRatio`, `AudioRatio`, and `AudioCompletionRatio` in `controller/option.go` and `model/option.go`.
- `controller.GetOptions` hides sensitive keys ending in `Token`, `Secret`, `Key`, `secret`, or `api_key` except explicit public keys in `isVisiblePublicKeyOption`.
- Keep `.env.example` as an example only; do not read or copy real `.env` values into planning docs or code comments.

## DB Mutation Patterns

- Use GORM for cross-database compatibility. `AGENTS.md` requires SQLite, MySQL, and PostgreSQL support.
- Use `model.DB` for business data and `model.LOG_DB` for logs. Both are initialized in `model/main.go`.
- Raw SQL must branch for PostgreSQL/MySQL/SQLite differences using `common.UsingPostgreSQL`, `common.UsingSQLite`, `common.UsingMySQL`, and the common column/boolean helpers initialized in `model/main.go`.
- Use GORM `AutoMigrate` and migration helper functions in `model/main.go`; avoid database-specific DDL without fallback.
- For concurrent task status changes, use compare-and-swap style mutations. `Task.UpdateWithStatus` is tested by `model/task_cas_test.go`.
- For async task/video billing, store immutable billing context in `model.Task.PrivateData.BillingContext` in `model/task.go`, then use `service.RefundTaskQuota`, `service.RecalculateTaskQuota`, and `service.RecalculateTaskQuotaByTokens` in `service/task_billing.go`.
- Token123 operational backup/change-log tables belong in PostgreSQL schema `token123_change_log` per `AGENTS.md`; do not create Token123 maintenance tables in the public business schema.

## Provider/Channel Conventions

- Register normal relay providers in `GetAdaptor` in `relay/relay_adaptor.go`; register task/video providers in `GetTaskAdaptor` in the same file.
- Add channel type constants and display names in `constant/channel.go` and API type constants in `constant/api_type.go` when introducing a new provider type.
- Normal channel packages should define `Adaptor`, `ModelList`, and `ChannelName`, following `relay/channel/claude/adaptor.go` and `relay/channel/codex/adaptor.go`.
- Task/video channel packages should define `TaskAdaptor`, provider request/response payload structs, `ModelList`, and `ChannelName`, following `relay/channel/task/doubao/adaptor.go` and `relay/channel/task/doubao/constants.go`.
- For new channels, check `StreamOptions` support and update `streamSupportedChannels` where applicable per `AGENTS.md`.
- Use `channel.SetupApiRequestHeader` and `channel.DoApiRequest` for standard relay HTTP behavior; use `channel.DoTaskApiRequest` for task adaptors.
- Header override and passthrough behavior is centralized in `relay/channel/api_request.go`; do not duplicate wildcard/regex/header-placeholder logic in provider packages.
- Preserve explicit zero values in upstream DTOs; tests in `dto/openai_request_zero_value_test.go` and `dto/gemini_generation_config_test.go` define the expected behavior.

## Token123/BytePlus Pricing Conventions

- `ModelRatio` is not a raw provider price; it participates in text billing as documented in `docs/token123-pricing-architecture.md` and implemented in `service/text_quota.go`.
- `CompletionRatio`, `CacheRatio`, `CreateCacheRatio`, `AudioRatio`, and `AudioCompletionRatio` are multiplier inputs to billing and display code, not independent global selling prices. See `relay/helper/price.go`, `service/text_quota.go`, and `model/pricing.go`.
- Token123 BytePlus/ModelArk changes should start from `docs/token123-byteplus-pricing-plan.md`; it explains why BytePlus cost tables are transformed before entering Token123 settings.
- Model metadata and vendor display are separate from pricing ratios. Pricing UI reads metadata from `model/pricing.go`, frontend hook `web/src/hooks/model-pricing/useModelPricingData.jsx`, and vendor defaults in `model/pricing_default.go`.
- When maintaining BytePlus video models, check `relay/channel/task/doubao/constants.go` for `ModelList` and video input ratios, `relay/channel/task/doubao/adaptor.go` for `EstimateBilling`, and `service/task_billing.go` for task billing logs and settlement.

## Frontend Conventions

- Use React 18, Vite, Semi UI, hooks, and feature folders under `web/src`.
- Keep data loading and table state in hooks such as `web/src/hooks/channels/useChannelsData.jsx` and `web/src/hooks/model-pricing/useModelPricingData.jsx`; keep rendering in components under `web/src/components`.
- Use the shared Axios instance `API` from `web/src/helpers/api.js` so auth headers, duplicate GET suppression, base URL, and global error handling remain consistent.
- Use `useTranslation()` and Chinese source keys with `t('...')`; frontend i18n is configured in `web/src/i18n/i18n.js` with fallback `zh-CN` and locale files under `web/src/i18n/locales`.
- Use Semi UI components from `@douyinfe/semi-ui` and existing table/modal patterns in `web/src/components/table/channels` and `web/src/components/table/model-pricing`.
- Preserve localStorage keys for table state and user preferences when refactoring hooks; examples include `channel-status-filter`, `page-size`, `channels-table-columns`, and `enable-tag-mode` in `web/src/hooks/channels/useChannelsData.jsx`.
- Use `bun` for frontend scripts. `AGENTS.md` and `web/package.json` prefer `bun install`, `bun run build`, `bun run lint`, and `bun run i18n:*`.

---

*Convention analysis: 2026-07-01*
