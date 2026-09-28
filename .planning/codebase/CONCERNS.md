> 2026-07-01 历史分析；当前精确源码与图谱基线见 [CURRENT-BASELINE.md](CURRENT-BASELINE.md)。以下事实须按当前源码复核。

# Codebase Concerns

**Analysis Date:** 2026-07-01

## Tech Debt

**Pricing semantics are split across runtime code, admin sync code, and local docs:**
- Issue: Token pricing is implemented in `relay/helper/price.go`, final text billing is in `service/text_quota.go`, per-call task pricing is in `relay/relay_task.go`, and upstream preset conversion is in `controller/ratio_sync.go`. The formulas are not centralized, and the Token123 pricing rules live in docs such as `docs/token123-pricing-architecture.md`, `docs/pricing-group-channel-ratio.md`, and `docs/token123-byteplus-pricing-plan.md`.
- Files: `relay/helper/price.go`, `service/text_quota.go`, `relay/relay_task.go`, `service/task_billing.go`, `controller/ratio_sync.go`, `docs/token123-pricing-architecture.md`, `docs/pricing-group-channel-ratio.md`, `docs/token123-byteplus-pricing-plan.md`
- Impact: BytePlus ModelArk changes can update `ModelRatio`, `CompletionRatio`, `CacheRatio`, `GroupRatio`, or `TaskPricePatches` with a correct-looking UI value but incorrect runtime quota. This is especially risky because `ModelRatio` is global by model name while routing is group/channel aware.
- Fix approach: Treat `service/text_quota.go` and `relay/helper/price.go` as the source of runtime truth. Add a shared pricing calculator for display, sync import, and runtime billing before adding more provider-specific conversion logic. Keep Token123 rollout SQL tied to formulas in `docs/token123-pricing-architecture.md`.

**Task/video billing has two pricing modes with implicit switches:**
- Issue: `relay/helper/price.go` uses `ModelPriceHelperPerCall` for task models, `relay/relay_task.go` multiplies `OtherRatios`, `service/task_polling.go` skips completion settlement when `BillingContext.PerCallBilling` is true, and `service/task_billing.go` recomputes token-based final quota when usage tokens exist.
- Files: `relay/helper/price.go`, `relay/relay_task.go`, `service/task_polling.go`, `service/task_billing.go`, `model/task.go`, `constant/env.go`, `common/init.go`
- Impact: A BytePlus video model can be accidentally billed as fixed per-call, pre-consumed ratio billing, or token-recalculated billing depending on `ModelPrice`, `ModelRatio`, `TaskPricePatches`, upstream `usage.total_tokens`, and `BillingContext.PerCallBilling`.
- Fix approach: For every new video alias, document whether it is per-call or token-recalculated. Add a focused test that submits the alias, stores `Task.PrivateData.BillingContext`, completes polling, and asserts final user quota, token quota, channel quota, and log `other`.

**Provider adaptors contain many intentional stubs:**
- Issue: Many relay adaptors return `errors.New("not implemented")` for unsupported modes. This is normal for a gateway, but it is spread across provider packages without a central capability matrix.
- Files: `relay/channel/ali/adaptor.go`, `relay/channel/aws/adaptor.go`, `relay/channel/claude/adaptor.go`, `relay/channel/deepseek/adaptor.go`, `relay/channel/vertex/adaptor.go`, `relay/channel/jimeng/adaptor.go`, `relay/channel/replicate/adaptor.go`
- Impact: Upstream merges that add a route or frontend option can expose an unimplemented provider path at runtime. This is most likely around Responses, image, audio, rerank, embedding, and task modes.
- Fix approach: Maintain a channel capability matrix keyed by `constant/channel.go` channel type and relay mode. New route additions should include tests against unsupported paths returning a controlled 4xx/501 instead of a generic relay failure.

**Large files concentrate behavior and merge conflicts:**
- Issue: Several high-change files mix parsing, validation, UI state, persistence, and business rules.
- Files: `controller/channel.go`, `controller/channel_upstream_update.go`, `controller/ratio_sync.go`, `relay/common/override.go`, `relay/channel/gemini/relay-gemini.go`, `service/channel_affinity.go`, `web/src/components/table/channels/modals/EditChannelModal.jsx`, `web/src/components/table/channels/modals/ParamOverrideEditorModal.jsx`, `web/src/pages/Setting/Ratio/UpstreamRatioSync.jsx`
- Impact: QuantumNous upstream syncs and Token123 local changes are likely to conflict in these files. Small fixes can unintentionally alter admin channel editing, model sync, ratio import, request override behavior, or Gemini billing.
- Fix approach: Before merging upstream into Token123 branches, diff these files first and run targeted tests for channel update, ratio sync, override, and task billing.

**JSON wrapper convention is not uniformly followed:**
- Issue: Project rules require `common.Marshal`, `common.Unmarshal`, and `common.DecodeJson`, but several files import `encoding/json`. Some usage is only for `json.RawMessage`; some files also perform decoding semantics close to business logic.
- Files: `AGENTS.md`, `controller/model_sync.go`, `controller/ratio_sync.go`, `model/task.go`, `service/task_billing_test.go`, `model/task_cas_test.go`
- Impact: New code can copy direct `encoding/json` calls into business paths and bypass project-wide JSON behavior.
- Fix approach: Keep `encoding/json` imports only for types such as `json.RawMessage`. Use `common.*` wrappers for marshal/unmarshal/decode in runtime logic and tests.

## Known Bugs

**Locale normalization rejects mixed-case zh locales:**
- Symptoms: `normalizeLocale` lowercases the input, then compares against mixed-case strings. `zh-CN` becomes `zh-cn` and does not match the `zh-CN` case.
- Files: `controller/model_sync.go`
- Trigger: Calling model/vendor sync with locale `zh-CN` or `zh-TW`.
- Workaround: Use default locale or patch `normalizeLocale` to compare lowercase constants and return canonical locale strings.

**Task token recalculation uses the same group as user group and using group:**
- Symptoms: `RecalculateTaskQuotaByTokens` loads `task.Group` and then calls `ratio_setting.GetGroupGroupRatio(group, group)`. Runtime text pricing uses `GetGroupGroupRatio(relayInfo.UserGroup, relayInfo.UsingGroup)` in `relay/helper/price.go`.
- Files: `service/task_billing.go`, `relay/helper/price.go`, `model/task.go`
- Trigger: A completed async task with `TotalTokens > 0`, `ModelRatio` billing, and a configured `GroupGroupRatio` where user group and using group differ.
- Workaround: Avoid relying on `GroupGroupRatio` for token-recalculated task/video billing until `TaskBillingContext` persists both `UserGroup` and `UsingGroup`.

**Suno task update can dereference a missing task map entry:**
- Symptoms: `updateSunoTasks` indexes `taskM[responseItem.TaskID]`, then calls `taskNeedsUpdate(task, responseItem)` without checking `task != nil`.
- Files: `service/task_polling.go`
- Trigger: Upstream returns a task ID that is not in the current polling batch map.
- Workaround: Add a nil guard before `taskNeedsUpdate`; log and skip unknown upstream IDs.

**Bulk task failure updates bypass CAS and refunds:**
- Symptoms: `TaskBulkUpdateByID` explicitly has no CAS guard. `TaskPollingLoop` uses it for tasks with empty upstream IDs, and `updateVideoTasks` / `updateSunoTasks` use it when channel lookup fails.
- Files: `model/task.go`, `service/task_polling.go`
- Trigger: A task transitions concurrently while a bulk failure path runs, or a channel lookup failure marks tasks failed without the normal `RefundTaskQuota` path.
- Workaround: For billing lifecycle transitions, load each task and call `UpdateWithStatus` plus `RefundTaskQuota` when needed.

## Security Considerations

**Production operational docs contain sensitive access details:**
- Risk: Several Token123 docs include production access context. Even without secret values, hostnames, usernames, container names, database names, and local key paths increase blast radius if committed or shared broadly.
- Files: `docs/token123-vendor-display-fix-20260620.md`, `docs/token123-deepseek-vendor-display-fix-20260620.md`, `docs/token123-model-icon-fix-20260621.md`, `docs/token123-change-log-schema-move-20260623.md`
- Current mitigation: The docs describe SQL backups and rollback procedures, and project rules forbid exposing API keys.
- Recommendations: Keep runbook details in private Rush docs where possible. In repo docs, refer to "production database" and "approved SSH key" without concrete host/key path values.

**Video proxy SSRF protection depends on runtime fetch settings:**
- Risk: `VideoProxy` fetches arbitrary task result URLs for non-Gemini/Vertex/OpenAI channels. It calls `ValidateURLWithFetchSetting`, but that returns success if `EnableSSRFProtection` is false.
- Files: `controller/video_proxy.go`, `common/ssrf_protection.go`, `setting/system_setting`
- Current mitigation: `common/ssrf_protection.go` has private/reserved IP checks, domain/IP lists, port checks, and optional DNS IP filtering.
- Recommendations: Keep SSRF protection enabled in production. For video providers, prefer provider-domain allowlists and enable `ApplyIPFilterForDomain`.

**Task private data persists upstream API keys for some video providers:**
- Risk: `model.InitTask` stores `TaskPrivateData.Key` for Gemini and Vertex tasks. `Task.PrivateData` is excluded from JSON responses, but it is stored in the database.
- Files: `model/task.go`, `controller/video_proxy.go`, `relay/relay_task.go`
- Current mitigation: `Task.PrivateData` has `json:"-"`, and `TaskModel2Dto` does not expose it.
- Recommendations: Treat task rows as secret-bearing. Avoid logging full tasks on failure paths; audit `logger.LogJson` calls around `service/task_polling.go`.

**Metadata passthrough can enforce or bypass costly video params:**
- Risk: Video task adaptors merge `metadata` into provider payloads. `taskcommon.UnmarshalMetadata` deletes `model`, but other price-driving fields such as `resolution`, `generate_audio`, `draft`, `duration`, or vendor-specific quality fields remain caller-controlled unless each adaptor validates them.
- Files: `relay/channel/task/taskcommon/helpers.go`, `relay/channel/task/doubao/adaptor.go`, `dto/video.go`, `dto/openai_video.go`, `relay/relay_task.go`
- Current mitigation: Model override through metadata is blocked.
- Recommendations: Do not publish low-price video aliases until the request path rejects or overwrites higher-cost parameters. Add adaptor-level allowlists for BytePlus/Seedance price dimensions.

## Performance Bottlenecks

**Task polling is serial inside each video channel:**
- Problem: `updateVideoTasks` loops task IDs one at a time and sleeps one second after each task.
- Files: `service/task_polling.go`
- Cause: Rate-limit avoidance is implemented as a fixed per-task sleep.
- Improvement path: Keep provider rate limits, but batch providers that support batch status APIs and add bounded per-channel concurrency for providers that allow it. Preserve CAS billing guards.

**Task polling scans unfinished tasks every 15 seconds:**
- Problem: `TaskPollingLoop` wakes every 15 seconds, reads up to `constant.TaskQueryLimit` unfinished tasks, groups them in memory, and processes each platform/channel.
- Files: `service/task_polling.go`, `model/task.go`
- Cause: Global polling loop with no shard key or worker lease.
- Improvement path: Add node ownership/lease or status update locks before running multiple production nodes. Index `progress`, `status`, `submit_time`, `platform`, and `channel_id` for production DBs.

**Upstream ratio/model sync can perform many network requests from admin actions:**
- Problem: `FetchUpstreamRatios` fetches up to eight upstreams concurrently and can call provider model endpoints, including OpenRouter with channel API keys.
- Files: `controller/ratio_sync.go`, `controller/model_sync.go`, `web/src/pages/Setting/Ratio/UpstreamRatioSync.jsx`
- Cause: Admin-triggered network sync pulls remote pricing/model metadata and parses large JSON payloads.
- Improvement path: Keep `maxRatioConfigBytes`, timeout, and concurrency caps. Add audit logs for imported upstream source, selected preset, and changed model count before applying settings.

## Fragile Areas

**BytePlus text pricing is easy to misconfigure:**
- Files: `relay/helper/price.go`, `service/text_quota.go`, `setting/ratio_setting/model_ratio.go`, `docs/token123-byteplus-pricing-plan.md`
- Why fragile: `ModelRatio` represents input token price scaling, `CompletionRatio` is output/input ratio, `CacheRatio` charges only reported cache-hit tokens, and `USDExchangeRate` changes display/conversion assumptions. BytePlus cost tiers require highest-tier input, output, cache, and audio dimensions to be calculated separately.
- Safe modification: Use the Token123 BytePlus runbook policy: target group `byteplus`, `GroupRatio=1`, display CNY, no fixed `ModelPrice` for text token models, cache-hit ratio from official cache-hit cost, and verification through repeated cache canaries before marketing cache behavior.
- Test coverage: Text quota tests exist in `service/text_quota_test.go`, but BytePlus-specific Dola/ModelArk pricing fixtures are not present.

**BytePlus/Doubao video aliases can undercharge if params are not enforced:**
- Files: `constant/channel.go`, `web/src/constants/channel.constants.js`, `relay/relay_adaptor.go`, `relay/channel/task/doubao/adaptor.go`, `relay/channel/task/doubao/constants.go`, `relay/relay_task.go`, `service/task_polling.go`
- Why fragile: Both `ChannelTypeVolcEngine=45` and `ChannelTypeDoubaoVideo=54` route to the Doubao task adaptor. The adaptor supports metadata-driven `resolution`, `duration`, `generate_audio`, `draft`, and video input discount detection. Current video input discount detection only looks at metadata `content` for `video_url`.
- Safe modification: Put Seedance/Doubao video on `ChannelTypeDoubaoVideo=54` unless the text `ChannelTypeVolcEngine=45` path is explicitly verified. Publish conservative highest-price aliases until lower-price aliases reject or override costlier params.
- Test coverage: No `relay/channel/task/doubao/*_test.go` is present.

**Model marketplace metadata is separate from route/channel availability:**
- Files: `model/model_meta.go`, `model/vendor_meta.go`, `controller/model_meta.go`, `controller/model_sync.go`, `docs/token123-vendor-display-fix-20260620.md`, `docs/token123-model-icon-fix-20260621.md`
- Why fragile: A public model needs channel routes, abilities, vendor row, model row, status, non-deleted metadata, and a non-empty description. Channel/abilities updates alone do not guarantee model marketplace visibility.
- Safe modification: For BytePlus rollout SQL, verify `vendors`, `models.vendor_id`, `models.status`, `models.deleted_at`, `channels.models`, `channels.model_mapping`, and `abilities.enabled`.
- Test coverage: Controller tests cover some token/channel behavior, but production metadata rollout SQL is verified manually.

**Param override can rewrite request bodies deeply:**
- Files: `relay/common/override.go`, `relay/common/override_test.go`, `relay/compatible_handler.go`, `relay/embedding_handler.go`, `relay/rerank_handler.go`, `web/src/components/table/channels/modals/ParamOverrideEditorModal.jsx`, `web/src/components/table/usage-logs/modals/ParamOverrideModal.jsx`
- Why fragile: Override supports set/delete/move/copy/prepend/append and conditional gjson checks. It is powerful enough to alter model params, headers, and provider-specific fields.
- Safe modification: For pricing-sensitive channels, review channel `ParamOverride` before assuming a rollout SQL or model alias enforces a price tier. Usage-log override audit should be checked during canary.
- Test coverage: `relay/common/override_test.go` is extensive, but provider-specific safety assertions need per-channel fixtures.

**Upstream sync divergence is an active merge hazard:**
- Files: `.git/config`, `AGENTS.md`, `go.mod`, `constant/channel.go`, `relay/relay_adaptor.go`, `controller/channel_upstream_update.go`, `controller/ratio_sync.go`, `web/src/constants/channel.constants.js`
- Why fragile: Local `main` tracks `origin/main`, the repo also tracks `quantum/main`, and `AGENTS.md` protects QuantumNous/new-api identity and module paths. Upstream adds channel types, pricing presets, video billing behavior, and frontend channel constants in the same files Token123 customizes.
- Safe modification: Sync in small batches. Resolve channel enum and frontend channel constants together. After every upstream sync, run focused tests for ratio sync, channel update, task billing, override, and video task conversion.
- Test coverage: Existing tests cover `controller/channel_upstream_update_test.go`, `controller/ratio_sync.go` helper behavior through `web` integration only partially, and task billing tests; there is no full upstream-sync regression suite.

## Scaling Limits

**Single-node task polling assumptions:**
- Current capacity: `TaskPollingLoop` processes up to `constant.TaskQueryLimit` unfinished tasks per 15-second loop and sweeps timed-out tasks in batches of 100.
- Files: `service/task_polling.go`, `model/task.go`
- Limit: Multiple app nodes can poll the same task. CAS reduces double settlement on terminal transitions, but non-terminal updates and bulk failure paths still use weaker guards.
- Scaling path: Add worker ownership or advisory locks before running task polling on multiple production nodes.

**Per-provider pricing data is global by model name:**
- Current capacity: `ModelRatio`, `CompletionRatio`, `CacheRatio`, and `ModelPrice` are keyed by model name.
- Files: `setting/ratio_setting/model_ratio.go`, `relay/helper/price.go`, `service/text_quota.go`, `docs/token123-pricing-architecture.md`
- Limit: The same local model name cannot have different `ModelRatio` values per upstream channel without using group ratios, group-group ratios, or distinct aliases.
- Scaling path: Add channel-scoped pricing or persist route-scoped effective pricing snapshots if Token123 needs different providers behind the same public alias.

## Dependencies at Risk

**External pricing metadata sources can overwrite local business policy:**
- Risk: Official presets and models.dev conversion import provider pricing into local ratio maps. models.dev chooses the cheapest non-zero duplicate provider entry.
- Impact: Token123's 2.2x resale policy and BytePlus max-tier policy can be overwritten by upstream cost metadata that is not group/channel aware.
- Migration plan: Treat `controller/ratio_sync.go` imports as suggestions only. Require a diff review that maps `model_name + using_group + selected_channel` before applying imported ratios.

**QuantumNous upstream adds new channel/video behavior in high-conflict files:**
- Risk: Upstream `quantum/main` contains current changes around Wan media mapping, Doubao Seedance safety/priority, 4K billing, and output resolution/video input billing.
- Impact: Token123 BytePlus video work can diverge from upstream billing rules or duplicate them under different aliases.
- Migration plan: Before local BytePlus changes, compare `relay/channel/task/doubao/*`, `relay/relay_task.go`, `service/task_polling.go`, `constant/channel.go`, and `web/src/constants/channel.constants.js` against `quantum/main`.

## Missing Critical Features

**No built-in BytePlus production rollout verifier:**
- Problem: BytePlus rollout depends on manual SQL plus docs. The app does not provide a verifier that checks options, channels, abilities, vendor metadata, model metadata, `/api/pricing`, and canary request logs together.
- Blocks: Safe repeated BytePlus ModelArk additions and rollbacks.
- Files: `docs/token123-byteplus-pricing-plan.md`, `docs/token123-pricing-architecture.md`, `controller/model_meta.go`, `controller/ratio_sync.go`, `model/model_meta.go`, `model/vendor_meta.go`

**No request-level enforcement for video price dimensions:**
- Problem: The code can forward metadata-driven video dimensions, but there is no central policy layer that rejects expensive parameters for a low-price alias.
- Blocks: Safe publication of 480p/720p/audio-off/draft aliases for Seedance/Doubao video.
- Files: `relay/channel/task/doubao/adaptor.go`, `relay/channel/task/taskcommon/helpers.go`, `dto/video.go`, `dto/openai_video.go`

**No route-scoped pricing model:**
- Problem: Runtime billing is model-name/global-ratio centric, while Token123's true cost is route-scoped by group and channel.
- Blocks: Clean support for one public model name routed to multiple upstream providers with different costs.
- Files: `relay/helper/price.go`, `service/text_quota.go`, `setting/ratio_setting/model_ratio.go`, `docs/pricing-group-channel-ratio.md`

## Test Coverage Gaps

**BytePlus/ModelArk text pricing fixtures:**
- What's not tested: Dola/Seed text model conversion from official USD/M cost to Token123 `ModelRatio`, `CompletionRatio`, and `CacheRatio`; cache-hit usage charging; audio price dimensions.
- Files: `service/text_quota.go`, `relay/channel/openai/relay-openai.go`, `setting/ratio_setting/model_ratio.go`
- Risk: A pricing change can look correct in the model marketplace but undercharge or overcharge runtime requests.
- Priority: High

**Doubao/Seedance video task pricing and parameter enforcement:**
- What's not tested: `EstimateBilling`, `video_input` ratio, metadata merge behavior, `resolution`/`duration`/`generate_audio`/`draft` enforcement, `TaskPricePatches`, and polling-stage token recalculation for Doubao.
- Files: `relay/channel/task/doubao/adaptor.go`, `relay/channel/task/doubao/constants.go`, `relay/relay_task.go`, `service/task_polling.go`, `service/task_billing.go`
- Risk: Production video aliases can undercharge high-cost requests or skip final settlement.
- Priority: High

**Task polling failure/refund edge cases:**
- What's not tested: Unknown upstream IDs in Suno polling, channel lookup failures, empty upstream task IDs, and CAS behavior combined with refund/settlement in the full polling loop.
- Files: `service/task_polling.go`, `service/task_billing.go`, `model/task.go`
- Risk: Users can be refunded incorrectly, not refunded, or double-settled during task lifecycle races.
- Priority: High

**Model/vendor metadata visibility checks:**
- What's not tested: End-to-end marketplace visibility after updating `vendors`, `models`, `channels`, `model_mapping`, and `abilities`.
- Files: `controller/model_meta.go`, `model/model_meta.go`, `model/vendor_meta.go`, `controller/model_sync.go`
- Risk: A model can be routable but missing from marketplace filters, or visible but not actually usable.
- Priority: Medium

**Upstream sync regression suite:**
- What's not tested: A scripted comparison and verification after merging `quantum/main` into Token123 local branches.
- Files: `constant/channel.go`, `relay/relay_adaptor.go`, `controller/channel_upstream_update.go`, `controller/ratio_sync.go`, `web/src/constants/channel.constants.js`, `web/src/pages/Setting/Ratio/UpstreamRatioSync.jsx`
- Risk: Upstream channel enum, frontend constants, and billing changes drift apart.
- Priority: Medium

---

*Concerns audit: 2026-07-01*
