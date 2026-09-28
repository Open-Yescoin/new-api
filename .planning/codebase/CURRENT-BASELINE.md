# 当前代码知识基线

**建立日期：** 2026-09-28（项目时区 Asia/Shanghai；文件扫描由本机 2026-09-27 America/Los_Angeles 执行）

## 来源与可验证范围

- 项目：Token123 的 `github.com/QuantumNous/new-api` 分支。本次扫描根：`/Users/lyz/.codex/worktrees/new-api/junxingchen-20260928`。
- 精确 Git 基线：`faeab5f5c66cd3b1d6d79489d1a79b86b40e7928`。此 SHA 仅描述已提交版本；暂存、未暂存和未跟踪文件的当前 SHA-256 另列于 `.planning/source-manifest.json` 的 `dirty_worktree_sha256`。
- `.planning/source-manifest.json` 的 `files_sha256` 逐个记录最终 Graphify 所扫描的 1018 个代码文件（包含本任务 7 个 SQL）的实际字节 SHA-256；同文件还记录 Graphify manifest 和 graph 的 SHA-256。扫描从上述 worktree 执行，排除构建产物 `web/dist`、依赖 `web/node_modules`、图谱产物 `graphify-out` 与规划资料 `.planning`。原先的临时代码在确认生产已有功能后撤回，2026-09-28 已重新执行 AST 增量扫描。
- 图谱路径：`graphify-out/graph.json`（7164 节点、26431 边）；同目录 `manifest.json` 为 1018 条相对扫描根的记录，`.graphify_root` 与 `.graphify_build.json` 记录扫描根和排除规则。此次为代码 AST 图谱；Markdown 文档、图片并未做语义图谱，因此不能用它声称这些内容已索引。
- 旧的 `.planning/codebase/{ARCHITECTURE,CONCERNS,CONVENTIONS,INTEGRATIONS,STACK,STRUCTURE,TESTING}.md` 均为 2026-07-01 快照，没有可验证的精确 commit 基线。本文件依据当前源码重建主要项目知识；旧文件的细节须按其所引源码在当前 SHA 再核，不直接视为当前事实。

## 当前结构

- 启动与配置：`main.go` 初始化 Gin、数据库和后台任务；`model/main.go` 通过 GORM 管理数据库；`model/option.go` 加载并更新持久化配置。`common/json.go` 提供项目要求的 JSON 包装函数。
- HTTP 路由：`router/` 注册 API、转发和前端路由；`controller/` 处理请求；`middleware/` 做认证、分组、渠道分配和限流；`relay/` 与 `relay/channel/` 实现协议转换与供应商适配；`service/` 负责计费等业务逻辑；`model/` 负责持久化。
- 前端：`web/src/` 使用 React 18、Vite 和 Semi Design，`web/src/i18n/locales/` 存放多语言 JSON。`web/package.json` 提供 Bun/Vite 的构建和检查脚本；`Dockerfile` 将已构建的 `web/dist` 复制进 Go 构建上下文。
- 数据库：`model/main.go` 的 GORM 层支持 SQLite、MySQL、PostgreSQL；业务迁移须兼容三者。Token123 生产配置使用 PostgreSQL；操作备份表归入 `token123_change_log` schema。

## 本任务所用调用与计费链

- `controller/relay.go` 接收中继请求，经 `middleware/distributor.go` 选择渠道，随后 `relay/channel/openai/`、`relay/channel/claude/` 等适配器调用上游。
- `model/channel.go` 保存渠道模型、分组与上游地址；`model/ability.go` 管理模型/分组/渠道的可路由关系。后台模型广场与价格由 `controller/pricing.go`、`model/pricing.go` 和 `web/src/components/table/model-pricing/` 展示。
- `setting/ratio_setting/model_ratio.go` 管理 ModelRatio、CompletionRatio、CacheRatio、CreateCacheRatio、ModelPrice 等；`relay/helper/price.go` 计算预扣；`service/text_quota.go` 用上游实际 usage 完成结算。已存在未提交的 `setting/ratio_setting/dynamic_text_pricing.go` 与对应调用改动，必须按其脏文件哈希对照，不归入 HEAD。
- 生产渠道的价格与路由设置不在 Git commit 中；数据库快照/回退 SQL 与真实请求日志才是上线验收依据。本任务的 524 主机切换 SQL 在 `docs/token123/junxingchen-20260928/sql/01-switch-api-host.sql`，为未提交文件。

## 测试与发布边界

- 后端使用 `go test` 进行包级验证；前端使用 Bun 的测试和 `bun run build`。当前任务有关测试位于 `service/text_quota_test.go`、`model/pricing_test.go`、`setting/ratio_setting/dynamic_text_pricing_test.go` 和 `web/src/helpers/modelPricing.test.js`。
- `.github/workflows/deploy.yml` 是当前 main 分支的生产 workflow；当前脚本使用单实例 `docker compose up -d --force-recreate new-api`，会重新创建唯一承流实例。生产不得中断，因此本任务发布前必须建立可验真的 Test 与无中断切换路径；不能从该脚本存在就推断可安全发布。
- 本机旧 checkout `/Users/lyz/new-api` 为另一个脏分支 `dd8f3f44e2e9b3bf1100898d0240eb7f79ec3dc2`，其 `.codegraph` SQLite 文件哈希索引与当前 worktree 不同。旧缓存不作为本次图谱来源，也不写入其他活动 worktree。

## 更新规则

最终快照以 `faeab5f5c66cd3b1d6d79489d1a79b86b40e7928` 为已提交源码基线；本任务的 SQL 和知识文件相对于此 SHA 的差异在 `dirty_worktree_sha256` 单列。Graphify 的 1018 个文件逐一校验，graph/manifest SHA 与扫描根均重新核验。后续任何文件编辑或合入须再次增量更新清单；本地 Git 基线不能代表生产容器版本。
