> 当前可验证源码与图谱基线见 [codebase/CURRENT-BASELINE.md](codebase/CURRENT-BASELINE.md)；以下为 2026-07-01 历史项目状态，不代表 2026-09-28 的部署版本。

# State: Token123 new-api Fork

**Updated:** 2026-07-01

## Project Reference

See: `.planning/PROJECT.md`

**Core value:** 所有模型上架、供应商接入和上游同步都必须保证价格、扣费、路由和回滚路径可验证。
**Current focus:** Phase 0: Project Intelligence Baseline

## Current Facts

- CodeGraph is initialized for `/Users/lyz/new-api`.
- GSD is being initialized as a brownfield project.
- Production `/home/ubuntu/new-api` baseline recorded in BytePlus docs: `a5bf5181`.
- Local repo HEAD at setup time: `e4093f4a5`.
- Official upstream remote `quantum/main` was fetched and compared during BytePlus video research.
- Current confirmed upstream gap: `origin/main..quantum/main = 254 commits`.

## Active Work

- BytePlus ModelArk Dola/Seedance pricing and rollout documentation lives under `/Users/lyz/Documents/Rush/docs/token123/byteplus-modelark-resale/`.
- Current safe direction for Seedance video: transplant minimal upstream video billing patches, then上架通用海外版模型；do not split low-price resolution aliases yet.

## Guardrails

- Do not revert unrelated user edits in the working tree.
- Do not run production SQL or restart production without explicit authorization.
- Do not copy official China-oriented video price table into BytePlus overseas pricing.
- Do not use `CompletionRatio`/`CacheRatio` as primary video billing controls.
- Do not allow upstream patches, pricing sync, or tiered billing changes to alter existing sold group ratios or model prices.

## Next Steps

1. Finish `.planning/codebase/` mapper outputs.
2. Re-run GSD status/progress once Phase 0 files exist.
3. Use CodeGraph before editing video billing code.
4. Open a scoped branch for upstream video patch transplant.

---
*Last updated: 2026-07-01 after CodeGraph/GSD initialization*
