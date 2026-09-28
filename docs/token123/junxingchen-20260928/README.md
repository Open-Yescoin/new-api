# Token123：君的星辰 2026-09-28 模型更新

## 目标与执行边界

将公开 OpenAI/Anthropic 目录限定为 4 款 GPT 文本、4 款 Claude 文本、2 款 GPT 图像。所有目标模型只在 `premium` 分组售卖；`GroupRatio.premium=0.5`。四个上游令牌组（gpt-pro、Claude特惠、Claude MAX、生图/视频）分别使用三枚互异令牌，每款模型对应三条独立启用渠道。

此次生产已有动态文本档位代码及公开价格展示，无需发布应用镜像。**执行 02 前必须先用正在运行的生产 `/api/pricing` 确认 `text_pricing` 两档存在，并核对实际计费能力；仅有数据库 `DynamicTextPricing` 选项并不证明运行时代码支持该规则。**当前发布工作流会重建唯一承流实例，故本次只执行可回退的 PostgreSQL 事务配置，不触碰运行中的容器。

## 定价

上游为人民币价目；按所选分组实际成本乘 2.2 售卖，不进行 USD/CNY 转换。Token123 的 `QuotaPerUnit=500000`、`premium=0.5`，所以文本标准档 `ModelRatio` 数字恰为每百万输入 token 的最终人民币售价。图像 `ModelPrice` 需为最终每张售价的两倍。

| 模型 | 上游分组 | 上游成本（输入/输出/缓存读/缓存写，¥/百万） | Token123 售价（同单位） | 配置 |
|---|---|---|---|---|
| gpt-5.6-sol | gpt-pro 0.4x | 2 / 12 / 0.2 / 2.5 | 4.4 / 26.4 / 0.44 / 5.5 | Ratio 4.4，Completion 6，Cache 0.1，CreateCache 1.25 |
| gpt-6-astra | gpt-pro 0.4x | 4 / 20 / 0.4 / 5 | 8.8 / 44 / 0.88 / 11 | 8.8，5，0.1，1.25 |
| gpt-5.6-terra | gpt-pro 0.4x | 0.8 / 4.8 / 0.08 / 1 | 1.76 / 10.56 / 0.176 / 2.2 | 1.76，6，0.1，1.25 |
| gpt-6-sol | gpt-pro 0.4x | 0.8 / 4 / 0.08 / 1 | 1.76 / 8.8 / 0.176 / 2.2 | 1.76，5，0.1，1.25 |
| claude-fable-5-1 | Claude特惠 0.1x | 1 / 5 / 0.025 / 2 | 2.2 / 11 / 0.055 / 4.4 | 2.2，5，0.025，2 |
| claude-opus-5-5 | Claude MAX 1.8x | 7.2 / 36 / 0.36 / 14.4 | 15.84 / 79.2 / 0.792 / 31.68 | 15.84，5，0.05，2 |
| claude-opus-5 | Claude特惠 0.1x | 0.5 / 2.5 / 0.05 / 1 | 1.1 / 5.5 / 0.11 / 2.2 | 1.1，5，0.1，2 |
| claude-sonnet-5 | Claude特惠 0.1x | 0.3 / 1.5 / 0.03 / 0.375 | 0.66 / 3.3 / 0.066 / 0.825 | 0.66，5，0.1，1.25 |

| 图像模型 | 上游成本（¥/张） | Token123 售价（¥/张） | `ModelPrice` |
|---|---:|---:|---:|
| gpt-image-2.5-flare | 0.12 | 0.264 | 0.528 |
| gpt-image-2.5-sunburst | 0.15 | 0.33 | 0.66 |

四款 GPT 在**整个请求的输入 token 数**达到 272001 后，输入与缓存读写价乘 2、输出价乘 1.5；272000 及以下保持标准价。公开 `/api/pricing` 显示 `junxing-gpt-pro-long-context` 档位。Opus 5.5 的 Claude MAX 上游只接受真实 Claude Code 客户端，渠道仅透传客户端实际 `User-Agent`；目录描述标明客户端限制。

## 生产变更与回退

按顺序执行 `sql/01-switch-api-host.sql`、`sql/02-refresh-nine-models.sql`、`sql/03-add-three-token-routes.sql`、`sql/04-route-gpt6-sol-legacy-host.sql`。四个文件均先以事务最终 `ROLLBACK` 演练，再真实提交。03 脚本以 psql 变量传入五枚密钥，仓库文件不保存密钥；新渠道为 49–53。04 把 `gpt-6-sol` 从新域名的三条混合 GPT 渠道移至旧域名专用渠道 57–59，复用同一组三枚不同 gpt-pro 令牌，其他三款 GPT 仍走新域名。备份全部保存在 `token123_change_log` schema，运行时 `public` 无操作备份表。回退按 **04 → 03 → 02 → 01** 逆序执行，先核对是否存在后续并发修改；04、03、02 的回退文件分别是 `sql/04-rollback-gpt6-sol-legacy-host.sql`、`sql/03-rollback-three-token-routes.sql`、`sql/02-rollback-nine-models.sql`。

审查后对 02 的软删除模型重用逻辑及 03 的 Opus 5.5 精确 ID 回退逻辑做了安全修订；这些修订发生在生产提交后，生产当前数据无需重放。生产另建 `token123_change_log.token123_junxing_three_token_model_20260928` 保存现有 Opus 5.5 模型 ID，以使修订后的 03 回退脚本与已提交状态一致。04 和 03 回退脚本均经生产事务 `ROLLBACK` 演练通过。

2026-09-28 生产核对：目标 10 款各有三条 `premium` 启用能力；各组抽样的三枚渠道密钥互异；OpenAI/Anthropic 公开目录恰好 10 款。五枚新渠道令牌 `GET /v1/models` 分别列出 Opus 5.5 或两款图像。上游控制台显示新建的 2 枚 MAX 与 1 枚图像令牌均已启用、无限额度、永不过期、无模型与 IP 限制。

## 验收与已知问题

- 524：2026-09-28 10:13:16 的 `/v1/responses` 请求在渠道 11 等待 132 秒后上游 524，额度 0；同用户随后 23 次 524。上游 FAQ 建议用 `api.centos.hk`，六条既有君的星辰文本渠道已切换；初期直连和 Token123 短请求成功。
- Claude：真实 Claude Code → Token123 → 渠道 51 调用 `claude-opus-5-5` 返回 `OK`，生产日志 122820 显示 2 输入、4 输出、2697 cache write、`premium=0.5`、`ModelRatio=15.84`。Fable 5.1、Opus 5、Sonnet 5 通过 `/v1/messages` 各得 HTTP 200，日志 122832–122834。
- 图像：Flare、Sunburst 通过 Token123 `/v1/images/generations` 各生成一张图，渠道 52、53，日志 122836、122839，额度分别 132000、165000，即 ¥0.264、¥0.33。
- GPT：Astra 和 Terra 经 Token123 `/v1/responses` 返回 200，日志 122814、122815。`gpt-5.6-sol` 有一次服务端 132 秒完成并计费（122816），但客户端 90 秒已超时；一枚令牌直连流式请求 75 秒无首字节，另一枚约 3 秒成功，证明上游存在明显波动。`gpt-6-sol` 在新域名经 Token123 127 秒后 503、额度 0（122817），两枚令牌直连约 3 秒返回 503，第三枚 20 秒无首字节；改测旧域名时三枚令牌均在约 3–8 秒完成流式响应。执行 04 后，经 Token123 非流式请求约 4.8 秒 HTTP 200，日志 122864 命中渠道 58，306 输入/5 输出、额度 291、`ModelRatio=1.76`、`premium=0.5`。旧域名仍存在历史 524 风险，需持续观察长请求；原 524 的 `gpt-5.6-sol` 不回迁旧域名。
- GPT 长档配置和公开显示已核对；尚未对当前价目做超过 272K token 的真实收费调用。生产历史日志 122064 等存在超过阈值的旧规则收费记录，只能作为动态计费机制的旁证。
