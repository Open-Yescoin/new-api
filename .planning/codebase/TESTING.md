> 2026-07-01 历史分析；当前精确源码与图谱基线见 [CURRENT-BASELINE.md](CURRENT-BASELINE.md)。以下事实须按当前源码复核。

# Testing Patterns

**Analysis Date:** 2026-07-01

## Test Framework

**Runner:**
- Go `testing` package is the primary backend runner.
- Assertions use `github.com/stretchr/testify` from `go.mod`, especially `require` and `assert`.
- Frontend unit/E2E test framework is not detected. `web/package.json` has build, Prettier, ESLint, and i18n scripts, but no Jest/Vitest/Playwright test script.

**Assertion Library:**
- Backend tests use `github.com/stretchr/testify/require` for fatal setup and invariants, and `github.com/stretchr/testify/assert` for nonfatal comparisons.
- JSON field existence tests use `github.com/tidwall/gjson`, for example `dto/openai_request_zero_value_test.go`.

**Run Commands:**
```bash
go test ./...                  # Run all Go tests
go test ./service              # Run one backend package
go test ./service -run Task    # Run matching tests in one package
cd web && bun run lint         # Prettier check for frontend
cd web && bun run eslint       # ESLint for JS/JSX
cd web && bun run build        # Frontend production build
```

## Test File Organization

**Location:**
- Backend tests are package-local `_test.go` files beside implementation files.
- Current root tree contains 31 Go test files outside `.claude/worktrees`.
- Duplicate tests also exist under `.claude/worktrees/new-api-auto-deploy`; ignore that worktree when assessing the main repository.
- Frontend test files are not detected under `web/src`.

**Naming:**
- Go tests use `TestFunctionOrBehavior_Case` for focused behavior: `TestGeneralOpenAIRequestPreserveExplicitZeroValues` in `dto/openai_request_zero_value_test.go`.
- Table tests use descriptive `name` fields and `t.Run(tt.name, ...)`, as in `setting/operation_setting/status_code_ranges_test.go` and `service/error_test.go`.
- DB integration tests use domain names such as `TestRefundTaskQuota_Wallet` and `TestUpdateWithStatus_ConcurrentWinner` in `service/task_billing_test.go` and `model/task_cas_test.go`.

**Structure:**
```text
common/*_test.go
controller/*_test.go
dto/*_test.go
model/*_test.go
relay/channel/*_test.go
relay/common/*_test.go
relay/helper/*_test.go
service/*_test.go
setting/*/*_test.go
```

## Existing Test Inventory

**DTO and request encoding:**
- `dto/openai_request_zero_value_test.go`: preserves explicit zero/false values in OpenAI chat and responses DTOs.
- `dto/gemini_generation_config_test.go`: preserves Gemini generation config zero values for camelCase and snake_case inputs.
- `dto/gemini_isstream_test.go`: tests Gemini stream detection with Gin request context.

**Relay and provider behavior:**
- `relay/channel/api_request_test.go`: header override, passthrough, runtime header override, and Codex pass-header behavior.
- `relay/channel/aws/relay_aws_test.go`: AWS relay behavior.
- `relay/channel/claude/relay_claude_test.go` and `relay/channel/claude/message_delta_usage_patch_test.go`: Claude response/usage details.
- `relay/channel/deepseek/adaptor_test.go`, `relay/channel/gemini/relay_gemini_usage_test.go`, and `relay/channel/minimax/adaptor_test.go`: provider-specific adaptor behavior.
- `relay/common/override_test.go`: param override operations such as `trim_prefix`, `replace`, `regex_replace`, `delete`, and runtime header operations.
- `relay/common/relay_info_test.go`: relay info behavior.
- `relay/common/stream_status_test.go`: stream status tracking.
- `relay/helper/stream_scanner_test.go`: streaming scanner edge cases, ordering, stop behavior, and high chunk counts.

**Service and billing behavior:**
- `service/text_quota_test.go`: text quota math, Claude/OpenRouter cache semantics, split cache creation ratios, and usage semantic detection.
- `service/task_billing_test.go`: async task refund and settlement against in-memory SQLite.
- `service/channel_affinity_template_test.go`: channel affinity override templates, skip retry, and Codex header passthrough.
- `service/channel_affinity_usage_cache_test.go`: cache usage observation across relay formats.
- `service/error_test.go`: status code remapping.
- `service/waffo_pancake_test.go`: Waffo Pancake payment behavior.

**Model and DB behavior:**
- `model/task_cas_test.go`: task snapshot equality and concurrent compare-and-swap update behavior.
- `model/commission_test.go`: commission model behavior.
- `model/payment_method_guard_test.go`: payment method guard behavior.

**Controller behavior:**
- `controller/channel_upstream_update_test.go`: upstream model update normalization, pending diff calculations, ignored regex patterns, notification throttling, and model mapping handling.
- `controller/payment_webhook_availability_test.go`: payment webhook availability behavior.
- `controller/token_test.go`: token controller behavior.
- `controller/topup_epay_guard_test.go`, `controller/topup_waffo_pancake_test.go`: top-up guards/payment behavior.

**Settings and utility behavior:**
- `setting/operation_setting/status_code_ranges_test.go`: parse/merge status code ranges, disable/retry checks, and legacy retry defaults.
- `setting/reasoning/suffix_test.go`: model suffix parsing.
- `common/url_validator_test.go`: URL validation.

## Test Structure

**Suite Organization:**
```go
func TestShouldSkipRetryAfterChannelAffinityFailure(t *testing.T) {
	tests := []struct {
		name string
		ctx  func() *gin.Context
		want bool
	}{ /* cases */ }

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, ShouldSkipRetryAfterChannelAffinityFailure(tt.ctx()))
		})
	}
}
```

**Patterns:**
- Use `t.Helper()` for local setup helpers such as `buildChannelAffinityTemplateContextForTest` in `service/channel_affinity_template_test.go` and `truncateTables` in `model/task_cas_test.go`.
- Use `gin.SetMode(gin.TestMode)`, `httptest.NewRecorder`, and `gin.CreateTestContext` for Gin-dependent logic in `service/channel_affinity_template_test.go`, `relay/channel/api_request_test.go`, and `relay/helper/stream_scanner_test.go`.
- Use `t.Cleanup` to delete DB rows or cache keys after each test, as in `service/task_billing_test.go` and `service/channel_affinity_template_test.go`.
- Use `t.Parallel()` for pure or isolated tests that do not mutate package globals, as in `relay/channel/api_request_test.go`, `relay/helper/stream_scanner_test.go`, and `service/error_test.go`.
- Avoid `t.Parallel()` around tests that mutate package globals such as `model.DB`, `common.UsingSQLite`, settings maps, or in-memory caches.

## Mocking

**Framework:** No dedicated mocking framework detected.

**Patterns:**
```go
rec := httptest.NewRecorder()
ctx, _ := gin.CreateTestContext(rec)
ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"gpt-5"}`))
ctx.Request.Header.Set("Content-Type", "application/json")
```

**What to Mock:**
- Use `httptest` and Gin test contexts for request/header/context behavior.
- Use in-memory SQLite through `github.com/glebarez/sqlite` for model/service tests that need persistence.
- Use local fake payloads and raw JSON for relay conversion, override, and DTO preservation tests.
- Use package caches directly when testing cache behavior, with `t.Cleanup` to delete keys. Example: `getChannelAffinityCache()` in `service/channel_affinity_template_test.go`.

**What NOT to Mock:**
- Do not mock `common.Marshal`/`common.Unmarshal`; they are part of the convention under test.
- Do not mock pricing math when testing Token123 billing. Construct `relaycommon.RelayInfo`, `types.PriceData`, or `model.TaskBillingContext` values and assert resulting quota/log fields.
- Do not hit real upstream providers, payment providers, Redis, or production databases in unit tests.

## Fixtures and Factories

**Test Data:**
```go
func seedUser(t *testing.T, id int, quota int) {
	t.Helper()
	user := &model.User{Id: id, Username: "test_user", Quota: quota, Status: common.UserStatusEnabled}
	require.NoError(t, model.DB.Create(user).Error)
}
```

**Location:**
- Fixtures are local helper functions inside package test files rather than shared fixture packages.
- DB seed helpers live in tests such as `service/task_billing_test.go` and `model/task_cas_test.go`.
- JSON fixtures are inline raw strings in tests such as `dto/openai_request_zero_value_test.go`, `relay/common/override_test.go`, and `service/text_quota_test.go`.

## DB Integration Tests

- Use `TestMain` to initialize in-memory SQLite for packages that need `model.DB`, as in `service/task_billing_test.go` and `model/task_cas_test.go`.
- Set `sqlDB.SetMaxOpenConns(1)` for in-memory SQLite tests to avoid separate empty databases per connection.
- Set global DB flags for test mode: `common.UsingSQLite = true`, `common.RedisEnabled = false`, `common.BatchUpdateEnabled = false`, and `common.LogConsumeEnabled = true`.
- Run `AutoMigrate` on only the tables needed by the package under test.
- Clean rows with `t.Cleanup` after each test, not at file load time.
- Use GORM operations in tests to match production DB compatibility constraints.

## Coverage

**Requirements:** No coverage threshold or coverage workflow is detected.

**View Coverage:**
```bash
go test ./... -cover
go test ./service -cover
```

## Test Types

**Unit Tests:**
- Pure parsing and normalization: `controller/channel_upstream_update_test.go`, `setting/reasoning/suffix_test.go`, `common/url_validator_test.go`.
- DTO marshal/unmarshal behavior: `dto/openai_request_zero_value_test.go`, `dto/gemini_generation_config_test.go`.
- Error/status behavior: `service/error_test.go`, `setting/operation_setting/status_code_ranges_test.go`.

**Integration Tests:**
- In-memory DB mutation tests: `service/task_billing_test.go`, `model/task_cas_test.go`, `model/commission_test.go`.
- Gin context/request integration: `relay/channel/api_request_test.go`, `service/channel_affinity_template_test.go`, `relay/helper/stream_scanner_test.go`.

**E2E Tests:**
- Not used. No browser or full-stack E2E framework is detected in `web/package.json`.

## Common Patterns

**Async Testing:**
```go
var wg sync.WaitGroup
wg.Add(goroutines)
for i := 0; i < goroutines; i++ {
	go func(idx int) {
		defer wg.Done()
		won, err := task.UpdateWithStatus(TaskStatusInProgress)
		if err == nil {
			wins[idx] = won
		}
	}(i)
}
wg.Wait()
```

**Error Testing:**
```go
_, err := ApplyParamOverride(input, override, nil)
if err == nil {
	t.Fatalf("expected error, got nil")
}
```

**JSON Assertions:**
```go
encoded, err := common.Marshal(req)
require.NoError(t, err)
require.True(t, gjson.GetBytes(encoded, "stream").Exists())
```

## Recommended Checks for Provider/Channel Changes

- Run `go test ./relay/... ./service ./dto` for relay provider changes.
- Add provider-specific tests beside the provider package when changing conversion, response parsing, usage mapping, or headers. Follow `relay/channel/gemini/relay_gemini_usage_test.go`, `relay/channel/deepseek/adaptor_test.go`, or `relay/channel/minimax/adaptor_test.go`.
- Add or update `relay/channel/api_request_test.go` when changing header overrides, header passthrough, runtime headers, placeholders, or channel test behavior.
- Add or update `relay/common/override_test.go` when changing param override operations, `pass_headers`, JSON path behavior, or runtime header side effects.
- Add DTO zero-value tests in `dto/*_test.go` when adding optional upstream request fields, especially scalar fields with `omitempty`.
- Verify new channel registration in `relay/relay_adaptor.go`, constants in `constant/channel.go` and `constant/api_type.go`, and frontend channel option display in `web/src/constants/channel.constants.js`.
- For streaming providers, run `go test ./relay/helper ./relay/common ./relay/channel/{provider}` and check `StreamOptions` support before adding the channel to stream-supported lists.

## Recommended Checks for Pricing Changes

- Run `go test ./service ./relay/helper ./model ./controller`.
- Add targeted cases in `service/text_quota_test.go` for text token billing, cache-read/cache-write semantics, Claude/OpenRouter usage semantics, and `ModelPrice` versus `ModelRatio` paths.
- Add targeted cases in `service/task_billing_test.go` for async task pre-consume, refund, subscription settlement, token quota refund, and `TaskBillingContext` log fields.
- Add or update controller/model option tests when changing `ModelRatio`, `ModelPrice`, `CompletionRatio`, `CacheRatio`, `CreateCacheRatio`, `ImageRatio`, `AudioRatio`, or `AudioCompletionRatio` validation/update behavior in `controller/option.go` and `model/option.go`.
- Build or inspect pricing UI after ratio changes with `cd web && bun run build`; the display path runs through `model/pricing.go` and `web/src/hooks/model-pricing/useModelPricingData.jsx`.
- For Token123 pricing maintenance, compare changes against `docs/token123-pricing-architecture.md`, `docs/pricing-group-channel-ratio.md`, and `docs/token123-byteplus-pricing-plan.md`.

## Recommended Checks for Video/Task Changes

- Run `go test ./service ./model ./relay/channel/task/...`.
- Add task adaptor tests for request validation, request payload conversion, response parsing, and polling behavior when changing `relay/channel/task/{provider}/adaptor.go`.
- For Doubao/BytePlus video input pricing, test `EstimateBilling` and `hasVideoInMetadata` behavior in `relay/channel/task/doubao/adaptor.go`, plus resulting `OtherRatios` logging in `service/task_billing.go`.
- Add or update `model/task_cas_test.go` when changing task status mutation, snapshots, task result URL compatibility, or concurrent polling logic.
- Add or update `service/task_billing_test.go` when changing `RefundTaskQuota`, `RecalculateTaskQuota`, `RecalculateTaskQuotaByTokens`, wallet/subscription behavior, token quota adjustment, or billing log `Other` fields.
- Verify `model.TaskPrivateData` and `model.TaskBillingContext` JSON/GORM behavior in `model/task.go` for fields needed by polling after process restarts.

## Recommended Checks for BytePlus ModelArk Maintenance

- Treat BytePlus text ModelArk channels as normal OpenAI-compatible relay changes unless a provider-specific adaptor is introduced; verify DTO zero values, usage parsing, cache token fields, and billing math.
- Check `docs/token123-byteplus-pricing-plan.md` before editing Token123 ratios; do not paste BytePlus cost values directly into `ModelRatio`/`CompletionRatio` without converting through the Token123 pricing formula.
- For BytePlus cache changes, add tests in `service/text_quota_test.go` or provider-specific relay tests that include cached token fields returned by the upstream response shape.
- For BytePlus video models, check `relay/channel/task/doubao/constants.go` for `ModelList` and `videoInputRatioMap`, `relay/channel/task/doubao/adaptor.go` for metadata conversion, and `service/task_billing.go` for quota reconciliation.
- Validate UI visibility for model/vendor/pricing updates through `model/pricing.go`, `model/pricing_default.go`, `web/src/hooks/model-pricing/useModelPricingData.jsx`, and `web/src/components/table/model-pricing/layout/PricingPage.jsx`.
- Run `go test ./...` before upstream sync, then `cd web && bun run lint && bun run build` for frontend-visible pricing/channel changes.

## Gaps

- No frontend unit tests are detected under `web/src`; React table, pricing, and channel workflows rely on build/lint and manual verification.
- No automated browser E2E tests are detected for admin pricing, channel creation, model test modals, or task/video submission.
- No CI workflow currently runs Go tests on pull requests; `.github/workflows/pr-check.yml` only enforces PR metadata/anti-slop checks.
- Coverage thresholds are not enforced.
- Cross-database behavior is mostly protected by conventions and GORM usage; tests use SQLite only. Raw SQL or migration changes still need explicit MySQL/PostgreSQL review.
- Provider/channel tests are uneven. Some providers have dedicated tests, while many adaptor packages rely on shared relay/common tests.
- BytePlus/ModelArk pricing and cache behavior is heavily documented in `docs/token123-byteplus-pricing-plan.md`, but dedicated automated tests should be added when changing concrete BytePlus response usage parsing or ratio formulas.

---

*Testing analysis: 2026-07-01*
