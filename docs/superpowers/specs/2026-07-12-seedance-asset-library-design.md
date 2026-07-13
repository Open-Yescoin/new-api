# Seedance 素材库网关设计

## 背景与结论

当前 Seedance 视频适配器可把公网图片 URL、视频 URL 和 `asset://<id>` 字符串透传给 BytePlus，但当前分支没有素材库 API。BytePlus 明确禁止 Seedance 2.0 直接使用含真人脸的普通 URL；真人必须先完成 H5 授权核身，获得 `LivenessFace` 私有素材组，再把同一人物的图片或视频登记到该组。因此，仅接入普通 AIGC `CreateAssetGroup` 不能满足“生成真人视频”的原始目标。

BytePlus 素材库不接收 multipart 文件。`CreateAsset` 接收一个公网可下载的 URL，由 BytePlus 异步拉取并处理素材。调用方必须先把本地文件上传到现有对象存储或 CDN，再调用素材库网关。

## 目标

1. 为普通 new-api Token 提供真人 H5 授权会话创建、核身结果查询，以及素材创建、列表、查询、更新和删除接口。
2. 图片与视频均可登记，分别使用 `AssetType: "Image"` 和 `AssetType: "Video"`。
3. 每个 new-api 用户绑定一个核身成功后由 BytePlus 创建的 `LivenessFace` 素材组；客户端不能指定或越过该分组。
4. `CreateAsset` 返回素材 ID；客户端轮询 `GetAsset` 至 `Status == "Active"`，随后在 Seedance 视频请求中使用 `asset://<id>`。
5. BytePlus Secret Key 只保存在系统设置中，不返回给普通用户或管理端浏览器；Access Key 可在管理页显示和更新。
6. 数据库迁移同时兼容 SQLite、MySQL 5.7.8+ 和 PostgreSQL 9.6+。

## 非目标

- new-api 不保存上传文件，也不新增对象存储或 multipart 上传能力。
- 不实现多出口、模板化协议转换、素材操作计费或独立限流；这些能力不属于当前用户流程。
- 不公开素材组 CRUD。真人组由 BytePlus H5 核身流程创建，new-api 只保存核身结果返回的组 ID。
- 不把普通 AIGC 虚拟人物素材组当作真人授权的替代品。

## API

所有接口使用 `middleware.TokenAuth()`，路径保持与 BytePlus Action 名称一致：

| 方法与路径 | 用途 |
| --- | --- |
| `POST /doubao/open/CreateVisualValidateSession` | 创建真人 H5 授权和核身会话 |
| `POST /doubao/open/GetVisualValidateResult` | 用 `BytedToken` 获取并绑定真人组 |
| `POST /doubao/open/CreateAsset` | 用公网 URL 登记图片或视频 |
| `POST /doubao/open/ListAssets` | 列出当前用户素材 |
| `POST /doubao/open/GetAsset` | 查询当前用户素材及处理状态 |
| `POST /doubao/open/UpdateAsset` | 更新当前用户素材元数据 |
| `POST /doubao/open/DeleteAsset` | 删除当前用户素材 |

客户端不能控制 `GroupId` 或 `ProjectName`。服务端始终用当前 Token 的用户 ID 查询绑定，并覆盖请求中的相关字段。

`CreateAsset` 至少校验：

- `URL` 必须为绝对 `http` 或 `https` URL；
- `AssetType` 只允许 `Image` 或 `Video`；
- 空 URL、私有协议和未知类型返回 HTTP 400。

## 组件边界

### 系统配置

`setting/system_setting/volc_asset.go` 定义单一 BytePlus 出口：Access Key、Secret Key、Region、Project Name、Group Type。Secret Key 采用只写更新语义：读取系统选项时返回空字符串，管理员提交空值时保留旧值。

`web/src/components/settings/VolcAssetSetting.jsx` 在当前系统设置页提供上述字段。页面只显示 Access Key、Region、Project Name、Group Type 以及“Secret Key 已配置”状态；Secret Key 输入框永不回填真实值。

### 用户分组绑定

`model/volc_asset_group.go` 保存 `user_id -> group_id`：

```text
volc_asset_user_groups
  id          primary key
  user_id     unique index
  group_id    text
  created_at  unix timestamp
  updated_at  unix timestamp
```

仅使用 GORM `Where`、`First`、`Create` 和事务，避免数据库方言相关 SQL。

真人授权时，服务层执行：

1. 调用 `CreateVisualValidateSession`，向客户端返回短期 `H5Link` 和 `BytedToken`；
2. 被拍摄者在 BytePlus H5 页面完成授权和真人核身；
3. 客户端用同一个 new-api 用户 Token 调用 `GetVisualValidateResult`；
4. 成功后保存 BytePlus 返回的 `GroupId`；重新核身会更新该用户绑定。

未绑定真人组的用户调用素材 CRUD 时返回 `409 asset_authorization_required`。

### BytePlus 客户端

`relay/channel/task/doubao/asset_client.go` 负责：

- 使用项目的 `common.Marshal` / `common.Unmarshal`；
- 构造 `Action`、`Version=2024-01-01` 请求；
- 生成 Volcengine HMAC-SHA256 签名；
- 限制上游响应体大小；
- 解析 `ResponseMetadata.Error` 与 `Result`；
- 返回结构化错误，不直接写 Gin 响应。

客户端通过小接口注入 handler，测试时使用假客户端，不发真实网络请求。

### HTTP 服务与归属校验

`relay/channel/task/doubao/asset_handler.go` 负责请求解析、当前用户识别、分组注入及响应转换。

- `ListAssets` 强制 `Filter.GroupIds = [用户分组]`。
- `CreateAsset` 强制 `GroupId = 用户分组`。
- `GetAsset`、`UpdateAsset`、`DeleteAsset` 先调用 `GetAsset`，验证返回的 `GroupId` 等于用户分组；不相等时返回 404。
- 上游业务错误保留 code/message；网络错误转换为 502；配置缺失返回 503。

### 视频生成引用

现有 `requestPayload.Content` 的 `image_url.url` 和 `video_url.url` 不限制 URL scheme，因此可直接透传 `asset://<id>`。新增适配器测试证明图片和视频素材引用不会被修改或丢弃，并更新调用文档，明确必须等待素材 `Active`。

## 数据流

```text
创建 H5 真人授权会话
  -> 被拍摄者授权并完成真人核身
  -> GetVisualValidateResult 返回并绑定 LivenessFace GroupId
  -> 本地文件
  -> 现有对象存储/CDN
  -> POST CreateAsset {URL, AssetType}
  -> 强制使用当前用户已核身的真人分组
  -> BytePlus 返回 asset id
  -> POST GetAsset 轮询至 Active
  -> POST /v1/video/generations，content 中使用 asset://<id>
  -> Seedance 异步生成视频
```

## 安全与错误处理

- 素材接口只接受 Token 鉴权，用户 ID 取自鉴权上下文，不接受客户端声明。
- 真人授权必须由被拍摄者在 BytePlus H5 页面明确完成，new-api 不绕过或代替核身。
- 任何素材详情、修改或删除前都做归属校验。
- AK/SK 不写日志；Secret Key 不返回浏览器；响应错误不包含签名头或凭证。
- 上游响应最大 10 MiB，避免无界读取。
- URL 校验只允许公网可表达的 HTTP(S) 形式；真正的网络可达性和素材格式由 BytePlus 异步处理结果决定。

## 测试与验收

1. 配置测试：序列化时 Secret Key 脱敏，空 Secret Key 更新保留旧值。
2. 模型测试：保存/读取用户绑定，重新授权后更新绑定。
3. 签名测试：固定时间下 canonical request 和 Authorization 稳定。
4. Service/Handler 测试：H5 授权结果绑定用户组；未授权返回 409；用户分组被强制注入；非法 URL/类型被拒绝；跨组 Get/Update/Delete 返回 404。
5. 适配器测试：`asset://` 图片和视频引用完整进入上游 `content`。
6. 路由测试：七个端点均注册并受 `TokenAuth` 保护。
7. 运行相关 Go 测试、全量 `go test ./...`，并对前端配置改动运行 Bun 构建。
