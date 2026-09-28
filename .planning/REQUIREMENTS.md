# Requirements: Token123 new-api Fork

**Defined:** 2026-07-01
**Core Value:** 所有模型上架、供应商接入和上游同步都必须保证价格、扣费、路由和回滚路径可验证。

## v1 Requirements

### Upstream Sync

- [ ] **UPSYNC-01**: 能记录当前生产 commit、本地 commit、origin/main、quantum/main，并在同步前明确差异范围。
- [ ] **UPSYNC-02**: 能按功能域筛选上游提交，避免将非目标功能的 254 个提交一次性合并到生产。
- [ ] **UPSYNC-03**: 每次移植上游补丁必须记录补丁来源、冲突处理、测试结果和代码回滚路径。

### BytePlus ModelArk

- [ ] **BYTE-01**: BytePlus 官方直连模型价格按官方 USD/M 乘固定 RMB 汇率和加价倍数核算，不额外走 `USDExchangeRate`。
- [ ] **BYTE-02**: 文本模型和视频模型的 channel type、路由、计费字段在文档和 SQL 中明确分开。
- [ ] **BYTE-03**: BytePlus 模型广场展示必须绑定正确 vendor、description、status、deleted_at 和 abilities。

### Video Tasks

- [ ] **VIDEO-01**: Seedance/Doubao 视频模型使用视频 channel type，例如 `ChannelTypeDoubaoVideo=54`。
- [ ] **VIDEO-02**: Seedance 2.0 动态计费按 `resolution` 和视频输入状态写入 `OtherRatios`。
- [ ] **VIDEO-03**: 视频任务验收必须确认上游或适配器提供 `usage.total_tokens > 0`，并触发 new-api 异步任务重算。
- [ ] **VIDEO-04**: 未能强制参数前，不发布 480p/720p/1080p/4K 低价别名。

### Production Change Control

- [ ] **PROD-01**: 每个生产 SQL 变更都有备份表、verify SQL、rollback SQL 和执行记录。
- [ ] **PROD-02**: 生产写库和重启前必须取得明确授权，授权文本包含目标 channel、模型、SQL 路径、备份表和重启动作。
- [ ] **PROD-03**: 部署失败、verify 失败、API 失败或扣费异常时，能区分代码回滚和 DB 回滚并逐项执行。
- [ ] **PROD-04**: 上游补丁前后必须 diff 既有 `GroupRatio` 和已售卖模型的 `ModelRatio`、`CompletionRatio`、`CacheRatio`、`CreateCacheRatio`、`ModelPrice`，除新增目标模型外不得变化。

### Project Intelligence

- [ ] **INTEL-01**: CodeGraph 在 `/Users/lyz/new-api` 可用，并能查询符号、调用链和影响面。
- [ ] **INTEL-02**: GSD `.planning/codebase/` 记录技术栈、架构、结构、规范、测试和风险。
- [ ] **INTEL-03**: GSD ROADMAP 记录上游同步和 BytePlus 上架的阶段化执行顺序。

## v2 Requirements

### Full Upstream Upgrade

- **UPGRADE-01**: 建立独立全量上游升级分支，覆盖前端、系统任务、日志、订阅、权限、DB 迁移和回归测试。
- **UPGRADE-02**: 为上游同步建立周期性差异报告和冲突预警。

### Video Productization

- **VIDPROD-01**: 支持按对外模型别名强制或拒绝 `metadata.resolution`、`generate_audio`、`draft`。
- **VIDPROD-02**: 支持安全上架低价分辨率别名，并有自动化测试覆盖价格漏洞。

## Out of Scope

| Feature | Reason |
|---------|--------|
| 直接全量合并 quantum/main | 当前差异 254 commits，风险过大 |
| 未验证 `total_tokens` 就生产上架视频按实际 token 收费 | new-api 结算依赖 `TotalTokens` |
| 未写 rollback 的生产变更 | 价格和模型路由错误会直接影响收入和可用性 |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| INTEL-01 | Phase 0 | In Progress |
| INTEL-02 | Phase 0 | In Progress |
| INTEL-03 | Phase 0 | In Progress |
| UPSYNC-01 | Phase 1 | Pending |
| UPSYNC-02 | Phase 1 | Pending |
| UPSYNC-03 | Phase 1 | Pending |
| BYTE-01 | Phase 2 | Pending |
| BYTE-02 | Phase 2 | Pending |
| BYTE-03 | Phase 3 | Pending |
| VIDEO-01 | Phase 2 | Pending |
| VIDEO-02 | Phase 2 | Pending |
| VIDEO-03 | Phase 2 | Pending |
| VIDEO-04 | Phase 3 | Pending |
| PROD-01 | Phase 3 | Pending |
| PROD-02 | Phase 3 | Pending |
| PROD-03 | Phase 3 | Pending |
| PROD-04 | Phase 2 | Pending |

**Coverage:**
- v1 requirements: 16 total
- Mapped to phases: 16
- Unmapped: 0

---
*Requirements defined: 2026-07-01*
*Last updated: 2026-07-01 after CodeGraph/GSD initialization*
