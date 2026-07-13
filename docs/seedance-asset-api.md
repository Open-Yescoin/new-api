# Seedance 2.0 真人素材库 API

Seedance 2.0 不接受未经授权的真人脸图片或视频作为普通公网参考素材。真人素材必须先完成 BytePlus H5 真人授权和核身，再登记到该真人的私有素材组；素材状态变为 `Active` 后，视频生成请求才能使用 `asset://<asset_id>`。

## 前置配置

管理员在管理后台的“Seedance 素材库”标签页配置：

- BytePlus Access Key
- BytePlus Secret Key（只写，不回显）
- Region，默认 `ap-southeast-1`
- Project Name，默认 `default`
- Group Type，真人场景固定使用 `LivenessFace`

素材接口使用 new-api Token：

```http
Authorization: Bearer <new-api-token>
Content-Type: application/json
```

BytePlus AK/SK 只由服务端用于签名，不交给客户端。

## 完整流程

### 1. 创建真人授权会话

```http
POST /doubao/open/CreateVisualValidateSession
```

```json
{
  "CallbackURL": "https://app.example.com/seedance/authorization/callback"
}
```

响应：

```json
{
  "BytedToken": "20260712000000...",
  "H5Link": "https://..."
}
```

前端应立即在手机浏览器打开或展示 `H5Link` 的二维码。被拍摄者需要登录 BytePlus、阅读并同意人脸信息处理与肖像授权条款，并完成真人核身。`H5Link` 和 `BytedToken` 都是短期凭证，过期后需重新创建会话。

### 2. 获取核身结果并绑定用户素材组

授权回调成功后，用同一个 new-api 用户 Token 调用：

```http
POST /doubao/open/GetVisualValidateResult
```

```json
{
  "BytedToken": "20260712000000..."
}
```

成功响应包含 `GroupId`。new-api 会把该真人 `GroupId` 绑定到当前 Token 所属用户，后续客户端不能自行指定或覆盖素材组。

如果核身尚未完成，上游可能返回 `ValidatePending`；稍后重试。核身失败或凭证过期时，应重新创建授权会话。

### 3. 把用户文件登记到真人素材库

BytePlus `CreateAsset` 不接受 multipart 文件。客户端需要先把本地图片或视频上传到现有对象存储/CDN，再提交公网 URL：

```http
POST /doubao/open/CreateAsset
```

图片：

```json
{
  "URL": "https://cdn.example.com/authorized-person/front.jpg",
  "AssetType": "Image"
}
```

视频：

```json
{
  "URL": "https://cdn.example.com/authorized-person/motion.mp4",
  "AssetType": "Video"
}
```

服务端会强制注入当前用户已核身的 `GroupId` 和管理员配置的 `ProjectName`。未完成真人授权时返回 HTTP `409 asset_authorization_required`。

同一素材组只能上传同一位已核身真人的素材。BytePlus 会进行人脸一致性和合规检查，不同人物、多人脸或质量不符合要求的素材可能入库失败。

### 4. 轮询素材状态

```http
POST /doubao/open/GetAsset
```

```json
{
  "Id": "asset-20260712-..."
}
```

只有 `Status` 或 `UpstreamStatus` 为 `Active` 时才可用于生成。`Processing` 时继续轮询，`Failed` 时向用户展示上游错误并重新上传符合要求的素材。

也可以列出当前用户的全部真人素材：

```http
POST /doubao/open/ListAssets
```

```json
{
  "PageNumber": 1,
  "PageSize": 20
}
```

### 5. 使用 `asset://` 生成真人视频

```http
POST /v1/video/generations
```

```json
{
  "model": "Seedance-2.0-海外版",
  "prompt": "图片1中的人物参考视频1的动作自然转身并介绍产品",
  "metadata": {
    "resolution": "720p",
    "ratio": "16:9",
    "duration": 8,
    "content": [
      {
        "type": "image_url",
        "image_url": { "url": "asset://asset-image-id" },
        "role": "reference_image"
      },
      {
        "type": "video_url",
        "video_url": { "url": "asset://asset-video-id" },
        "role": "reference_video"
      }
    ]
  }
}
```

`asset://` 真人素材应仅用于 Seedance 2.0 标准版或 Fast 版。提示词中按输入顺序使用“图片1”“视频1”等称呼，不要要求模型直接理解素材 ID。

## 其他管理接口

| 路径 | 用途 |
| --- | --- |
| `POST /doubao/open/ListAssets` | 列出当前用户素材 |
| `POST /doubao/open/GetAsset` | 查询素材和处理状态 |
| `POST /doubao/open/UpdateAsset` | 更新素材名称等元数据 |
| `POST /doubao/open/DeleteAsset` | 删除素材 |

`GetAsset`、`UpdateAsset` 和 `DeleteAsset` 都会先校验素材是否属于当前用户绑定的真人组；跨用户素材统一返回 404。

## 合规要求

- 必须取得真人明确授权，不能代替他人完成核身。
- 平台应保留授权、素材和最终使用者之间的可追溯关系。
- 用户重新完成真人授权后，新 `GroupId` 会替换该 new-api 用户原有绑定。
- 管理员必须确认 BytePlus 账号已开通真人私有素材库及相应高级创作权益。
