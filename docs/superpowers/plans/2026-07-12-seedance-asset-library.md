# Seedance Asset Library Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a token-authenticated, per-user-isolated BytePlus Seedance asset library gateway for image and video references.

**Architecture:** A focused Volcengine Asset client signs upstream Action requests with the configured AK/SK. A service layer starts BytePlus H5 real-person authorization, binds the returned LivenessFace group to the authenticated new-api user, and forces every asset operation into that verified group; thin Gin handlers expose two authorization and five asset actions. The existing Seedance converter continues to forward `asset://<id>` references unchanged.

**Tech Stack:** Go 1.22+, Gin, GORM v2, SQLite/MySQL/PostgreSQL, React 18, Semi Design, i18next, Bun.

---

## Scope correction from authoritative provider evidence

During implementation, current BytePlus documentation confirmed that Seedance 2.0 rejects ordinary URL uploads containing real human faces. The original AIGC auto-provisioning design therefore cannot satisfy the stated real-person objective. Tasks below are executed with this corrected invariant: `CreateVisualValidateSession -> GetVisualValidateResult -> LivenessFace GroupId -> CreateAsset -> Active -> asset://id`. AIGC group auto-creation is excluded.

### Task 1: Safe Volcengine asset configuration

**Files:**
- Create: `setting/system_setting/volc_asset.go`
- Create: `setting/system_setting/volc_asset_test.go`
- Modify: `model/option.go`
- Modify: `controller/option.go`
- Test: `controller/option_volc_asset_test.go`

- [ ] **Step 1: Write failing configuration tests**

Test that defaults are `ap-southeast-1`, `LivenessFace`, and the BytePlus Ark host; test that an empty incoming `secret_key` preserves the existing value; test that `GetOptions` returns `secret_key_configured: true` without returning the secret.

```go
func TestMergeVolcAssetSettingsPreservesSecret(t *testing.T) {
    current := VolcAssetSettings{SecretKey: "existing"}
    incoming := VolcAssetSettings{AccessKey: "ak", SecretKey: ""}
    got := MergeVolcAssetSettings(current, incoming)
    require.Equal(t, "existing", got.SecretKey)
}
```

- [ ] **Step 2: Run tests and confirm the missing-type/function failures**

Run: `go test ./setting/system_setting ./controller -run 'VolcAsset' -count=1`

Expected: FAIL because the configuration helpers and option integration do not exist.

- [ ] **Step 3: Implement configuration and option persistence**

Define:

```go
type VolcAssetSettings struct {
    AccessKey   string `json:"access_key"`
    SecretKey   string `json:"secret_key,omitempty"`
    Region      string `json:"region"`
    ProjectName string `json:"project_name"`
    GroupType   string `json:"group_type"`
}

type PublicVolcAssetSettings struct {
    AccessKey          string `json:"access_key"`
    SecretKeyConfigured bool  `json:"secret_key_configured"`
    Region             string `json:"region"`
    ProjectName        string `json:"project_name"`
    GroupType          string `json:"group_type"`
}
```

Add `VolcAssetConfig` to `InitOptionMap`; normalize and merge it before `model.UpdateOption` writes the database value; parse it in `updateOptionMap`. Special-case `controller.GetOptions` to marshal `PublicVolcAssetSettings` instead of exposing the stored JSON.

- [ ] **Step 4: Re-run focused tests**

Run: `go test ./setting/system_setting ./controller -run 'VolcAsset' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the configuration slice**

```bash
git add setting/system_setting/volc_asset.go setting/system_setting/volc_asset_test.go model/option.go controller/option.go controller/option_volc_asset_test.go
git commit -m "feat: add safe Seedance asset settings"
```

### Task 2: Cross-database user asset-group binding

**Files:**
- Create: `model/volc_asset_group.go`
- Create: `model/volc_asset_group_test.go`
- Modify: `model/main.go`

- [ ] **Step 1: Write failing repository tests**

Use in-memory SQLite and verify missing bindings return `gorm.ErrRecordNotFound`, save/read round trips work, and a second successful authorization updates the same user's binding without creating a duplicate row.

```go
func TestSaveAndGetVolcAssetUserGroup(t *testing.T) {
    setupVolcAssetTestDB(t)
    require.NoError(t, SaveVolcAssetUserGroup(42, "group-42"))
    got, err := GetVolcAssetUserGroup(42)
    require.NoError(t, err)
    require.Equal(t, "group-42", got.GroupId)
}
```

- [ ] **Step 2: Run the repository tests and confirm RED**

Run: `go test ./model -run 'VolcAssetUserGroup' -count=1`

Expected: FAIL because the model and repository functions do not exist.

- [ ] **Step 3: Implement the model and migrations**

Create a GORM model with `UserId` as a unique index and `GroupId` as `type:text`. Implement `GetVolcAssetUserGroup` with `Where(...).First` and `SaveVolcAssetUserGroup` with a create-or-update flow plus a re-read after a unique constraint race. Add the model to both normal and fast `AutoMigrate` lists in `model/main.go`.

- [ ] **Step 4: Run focused repository tests**

Run: `go test ./model -run 'VolcAssetUserGroup' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the persistence slice**

```bash
git add model/volc_asset_group.go model/volc_asset_group_test.go model/main.go
git commit -m "feat: isolate Seedance assets by user group"
```

### Task 3: Signed BytePlus Asset client

**Files:**
- Create: `relay/channel/task/doubao/asset_client.go`
- Create: `relay/channel/task/doubao/asset_client_test.go`

- [ ] **Step 1: Write failing signing and response tests**

Inject a fixed UTC clock. Assert the request includes `X-Date`, `X-Content-Sha256`, and an Authorization credential scope of `<date>/ap-southeast-1/ark/request`. Use `httptest.Server` to verify `Action` and `Version=2024-01-01`, successful `Result` decoding, and `ResponseMetadata.Error` conversion.

```go
client := NewVolcAssetClient(cfg, httpClient, func() time.Time {
    return time.Date(2026, 7, 12, 1, 2, 3, 0, time.UTC)
})
err := client.Call(context.Background(), "ListAssets", ListAssetsRequest{}, &result)
```

- [ ] **Step 2: Run client tests and confirm RED**

Run: `go test ./relay/channel/task/doubao -run 'VolcAssetClient' -count=1`

Expected: FAIL because the client does not exist.

- [ ] **Step 3: Implement the client**

Define a narrow interface and typed upstream error:

```go
type AssetAPI interface {
    Call(ctx context.Context, action string, request any, result any) error
}

type AssetAPIError struct {
    StatusCode int
    Code       string
    Message    string
}
```

Use `common.Marshal`, `common.Unmarshal`, `common.Sha256Raw`, and `common.HmacSha256Raw`. Read at most 10 MiB, never log credentials, and return `ErrVolcAssetNotConfigured` when AK or SK is missing.

- [ ] **Step 4: Run focused client tests**

Run: `go test ./relay/channel/task/doubao -run 'VolcAssetClient' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the client slice**

```bash
git add relay/channel/task/doubao/asset_client.go relay/channel/task/doubao/asset_client_test.go
git commit -m "feat: add signed BytePlus asset client"
```

### Task 4: User-isolated asset service

**Files:**
- Create: `relay/channel/task/doubao/asset_service.go`
- Create: `relay/channel/task/doubao/asset_service_test.go`

- [ ] **Step 1: Write failing service tests**

Provide fake `AssetAPI` and `AssetGroupRepository` implementations. Verify first use creates `newapi-user-42`, later use reuses the binding, list/create overwrite client group fields, unsupported schemes and asset types fail validation, and cross-group get/update/delete return `ErrAssetNotFound`.

```go
service := NewAssetService(fakeAPI, fakeGroups)
_, err := service.CreateAsset(ctx, 42, CreateAssetRequest{
    URL: "https://cdn.example/person.jpg", AssetType: "Image",
})
require.NoError(t, err)
require.Equal(t, "group-42", fakeAPI.lastCreate.GroupId)
```

- [ ] **Step 2: Run service tests and confirm RED**

Run: `go test ./relay/channel/task/doubao -run 'AssetService' -count=1`

Expected: FAIL because the service does not exist.

- [ ] **Step 3: Implement service DTOs and invariants**

Define request/response DTOs for the two real-person authorization actions and five asset actions. `GetVisualValidateResult` stores the verified `GroupId`; asset calls without a verified binding return `ErrAssetGroupNotAuthorized`. For ownership checks call `GetAsset` and compare `GroupId`. Accept only absolute HTTP(S) URLs and `Image`/`Video` asset types.

- [ ] **Step 4: Run focused service tests**

Run: `go test ./relay/channel/task/doubao -run 'AssetService' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the service slice**

```bash
git add relay/channel/task/doubao/asset_service.go relay/channel/task/doubao/asset_service_test.go
git commit -m "feat: enforce Seedance asset ownership"
```

### Task 5: Token-authenticated HTTP endpoints

**Files:**
- Create: `relay/channel/task/doubao/asset_handler.go`
- Create: `relay/channel/task/doubao/asset_handler_test.go`
- Create: `controller/asset.go`
- Modify: `router/video-router.go`

- [ ] **Step 1: Write failing handler tests**

Construct Gin test contexts with `c.Set("id", 42)`. Verify invalid JSON returns 400, validation errors return 400, unknown ownership returns 404, missing configuration returns 503, upstream transport errors return 502, and successful action results preserve upstream JSON fields.

- [ ] **Step 2: Run handler tests and confirm RED**

Run: `go test ./relay/channel/task/doubao -run 'AssetHandler' -count=1`

Expected: FAIL because the handler does not exist.

- [ ] **Step 3: Implement handlers, controller wrappers, and routes**

Expose the two authorization actions and five user asset actions under a `/doubao` group protected by `middleware.TokenAuth()`:

```go
doubaoGroup.POST("/open/CreateVisualValidateSession", controller.RelayCreateVisualValidateSession)
doubaoGroup.POST("/open/GetVisualValidateResult", controller.RelayGetVisualValidateResult)
doubaoGroup.POST("/open/ListAssets", controller.RelayListAssets)
doubaoGroup.POST("/open/GetAsset", controller.RelayGetAsset)
doubaoGroup.POST("/open/CreateAsset", controller.RelayCreateAsset)
doubaoGroup.POST("/open/UpdateAsset", controller.RelayUpdateAsset)
doubaoGroup.POST("/open/DeleteAsset", controller.RelayDeleteAsset)
```

Use `common.DecodeJson` or `common.UnmarshalBodyReusable`; do not import `encoding/json` for marshaling.

- [ ] **Step 4: Run handler and router package tests**

Run: `go test ./relay/channel/task/doubao ./router -run 'Asset|Doubao' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the HTTP slice**

```bash
git add relay/channel/task/doubao/asset_handler.go relay/channel/task/doubao/asset_handler_test.go controller/asset.go router/video-router.go
git commit -m "feat: expose Seedance asset library API"
```

### Task 6: Prove Seedance uses active asset references

**Files:**
- Modify: `relay/channel/task/doubao/adaptor_test.go`
- Create: `docs/seedance-asset-api.md`

- [ ] **Step 1: Add failing converter tests**

Add one image and one video case using `asset://image-id` and `asset://video-id` in `metadata.content`; assert `convertToRequestPayload` retains the scheme, type, and role. If these pass immediately, temporarily mutate the expected input to prove the assertion detects a changed reference, then restore it.

- [ ] **Step 2: Run converter tests**

Run: `go test ./relay/channel/task/doubao -run 'AssetReference' -count=1`

Expected: PASS on the existing converter after the assertion has been proven sensitive.

- [ ] **Step 3: Add end-to-end API documentation**

Document: create H5 authorization session; bind `GroupId` after verification; upload local file to existing CDN; call `CreateAsset`; poll `GetAsset` until `Active`; submit `/v1/video/generations` using `asset://<id>` as `image_url.url` or `video_url.url`; never use a Pending asset.

- [ ] **Step 4: Commit tests and docs**

```bash
git add relay/channel/task/doubao/adaptor_test.go docs/seedance-asset-api.md
git commit -m "docs: explain Seedance asset generation flow"
```

### Task 7: Admin configuration UI and translations

**Files:**
- Create: `web/src/components/settings/VolcAssetSetting.jsx`
- Modify: `web/src/pages/Setting/index.jsx`
- Modify: `web/src/i18n/locales/zh.json`
- Modify: `web/src/i18n/locales/en.json`
- Modify: `web/src/i18n/locales/fr.json`
- Modify: `web/src/i18n/locales/ru.json`
- Modify: `web/src/i18n/locales/ja.json`
- Modify: `web/src/i18n/locales/vi.json`

- [ ] **Step 1: Add a focused UI regression test or static contract check**

Add a Bun-executed source contract test following existing `web/src/*regression.test.mjs` patterns. Assert the settings component sends one `VolcAssetConfig` JSON option, leaves `secret_key` empty unless the admin enters a replacement, and displays the configured-secret status.

- [ ] **Step 2: Run the UI test and confirm RED**

Run the matching script with Bun from `web/`.

Expected: FAIL because the component is absent.

- [ ] **Step 3: Implement the settings tab**

Create a Semi Design form for Access Key, write-only Secret Key, Region (`ap-southeast-1` default), Project Name, and Group Type (`LivenessFace`). Add a dedicated `Seedance 素材库` tab to the existing settings page and save through `PUT /api/option/` with key `VolcAssetConfig`.

- [ ] **Step 4: Synchronize and lint translations**

Run: `bun run i18n:sync && bun run i18n:lint`

Expected: exit 0 with all six locale files aligned.

- [ ] **Step 5: Build the frontend**

Run: `bun run build`

Expected: exit 0.

- [ ] **Step 6: Commit the UI slice**

```bash
git add web/src/components/settings/VolcAssetSetting.jsx web/src/pages/Setting/index.jsx web/src/i18n/locales/*.json web/src/*volc-asset*test.mjs
git commit -m "feat: configure Seedance asset library"
```

### Task 8: Completion verification

**Files:**
- Verify all files above.

- [ ] **Step 1: Run formatting and diff checks**

Run: `gofmt -w` on changed Go files, then `git diff --check`.

Expected: no formatting or whitespace errors.

- [ ] **Step 2: Run focused backend tests**

Run: `go test ./setting/system_setting ./model ./controller ./relay/channel/task/doubao ./router -count=1`

Expected: all packages pass.

- [ ] **Step 3: Run full backend tests**

Run: `go test ./... -count=1`

Expected: all packages pass. Any pre-existing unrelated failure must be recorded separately and must not be presented as feature success.

- [ ] **Step 4: Run frontend validation**

Run from `web/`: `bun run i18n:lint && bun run build`

Expected: exit 0.

- [ ] **Step 5: Audit the objective against evidence**

Confirm each requirement has direct evidence: URL registration, image/video types, Active polling contract, `asset://` generation passthrough, token auth, per-user isolation, secret redaction, cross-DB migration, tests, and docs.
