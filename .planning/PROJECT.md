# Token123 new-api Fork

## What This Is

Token123 的 new-api 私有分支，用于管理模型供应商、渠道、价格、模型广场展示和 API 转发。它基于上游 `QuantumNous/new-api`，但生产分支已经有本地定制和上游提交差异，后续维护重点是可控地同步上游能力，同时保护生产计费和供应商配置。

## Core Value

所有模型上架、供应商接入和上游同步都必须保证价格、扣费、路由和回滚路径可验证。

## Business Context

- **Customer**: Token123 用户和后台运营人员
- **Revenue model**: 按模型调用 token 或任务用量加价售卖
- **Success metric**: 模型可用、价格正确、扣费不亏损、异常可快速回滚
- **Strategy notes**: BytePlus ModelArk 调研文档在 `/Users/lyz/Documents/Rush/docs/token123/byteplus-modelark-resale/`

## Requirements

### Validated

- ✓ new-api 可通过渠道、模型映射、分组和倍率配置管理多供应商模型。
- ✓ Token123 生产目前以 `origin/main` 对齐的部署提交为基线维护。

### Active

- [ ] BytePlus ModelArk 官方直连模型价格必须按官方 USD/M 换算人民币售卖价，不走中转站倍率假设。
- [ ] Seedance/Doubao 视频模型必须使用视频任务通道和异步任务结算链路，不混用文本 channel shape。
- [ ] 上游 `QuantumNous/new-api` 的视频相关补丁应以最小变更移植，避免一次性全量合并 254 个提交。
- [ ] 每次生产写库、重启、代码部署都必须有备份、verify、rollback 和明确授权记录。
- [ ] 代码图谱和 GSD 文档应为后续上游同步、冲突审查、生产 runbook 提供稳定上下文。

### Out of Scope

- 直接全量升级到 `QuantumNous/new-api` 最新 main — 风险包含前端、系统任务、日志、订阅、权限和 DB 行为变化，应另开升级项目。
- 在未强制 `resolution`、`generate_audio`、`draft` 前上架低价视频别名 — 会形成低价请求高成本规格的漏洞。
- 在未验证 BytePlus 返回 `usage.total_tokens > 0` 前按实际视频 token 做生产承诺 — new-api 结算依赖 `TotalTokens`。

## Context

- 本地仓库：`/Users/lyz/new-api`
- 生产仓库路径：`/home/ubuntu/new-api`
- 生产基线：`a5bf5181`
- 本地当前 HEAD：`e4093f4a5`
- 官方上游 remote：`quantum https://github.com/QuantumNous/new-api.git`
- 当前确认差距：`origin/main..quantum/main = 254 commits`
- CodeGraph 已初始化在 `.codegraph/`，用于快速查询符号、调用关系和影响面。
- GSD 以 brownfield 项目初始化，先生成 `.planning/codebase/`，再维护需求和路线图。

## Constraints

- **Production safety**: 任何生产 SQL 或重启必须先有明确授权、备份表、verify SQL、rollback SQL。
- **Pricing**: BytePlus 官方直连价格已经换算 RMB 时，不再额外乘站点 `USDExchangeRate`。
- **Sold price immutability**: 已售卖分组倍率和模型售价是产品价格；上游更新不得让 0.3/0.5/1 倍分组或已上架模型的输入/输出/缓存价格漂移。
- **Video billing**: 视频任务以 `ModelRatio * GroupRatio * OtherRatios` 和 `usage.total_tokens` 重算为核心，不依赖 `CompletionRatio`/`CacheRatio`。
- **Upstream sync**: 不做无计划全量升级；先定位相关提交、评估冲突、测试、再部署。
- **Working tree**: 仓库已有用户修改和未跟踪文件，维护任务不得回滚无关改动。

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| 初始化 CodeGraph | 项目较大且上游差异多，需要符号级查询支撑后续同步 | ✓ Good |
| 初始化 GSD brownfield 文档 | 后续上游更新和生产变更需要项目记忆、需求和路线图 | — Pending |
| Seedance 先移植视频补丁再上架 | 当前生产代码缺少 Seedance 2.0 分辨率动态计费 | — Pending |
| 不拆低价分辨率别名 | 现有视频路径不能强制请求参数 | ✓ Good |
| 上游更新不得改既有价格 | 分组倍率和已上架模型售价是对用户售卖的产品 | ✓ Good |

---
*Last updated: 2026-07-01 after CodeGraph/GSD initialization for upstream sync maintenance*
