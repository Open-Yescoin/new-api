# Seedance User Authorization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a safe Token123 console flow for BytePlus H5 liveness authorization, mobile QR handoff, user-bound `LivenessFace` assets, and `asset://` references.

**Architecture:** Keep `/doubao/open/*` API-token routes compatible and add `/api/seedance/assets/*` dashboard routes protected by `UserAuth`. A new service persists only a SHA-256 digest of each short-lived `BytedToken`, checks that it belongs to the current user, and atomically saves the returned `GroupId`; React implements the approved authorization state machine and asset CRUD without exposing credentials or group IDs.

**Tech Stack:** Go 1.22, Gin, GORM v2, React 18, Semi Design, `qrcode.react`, Bun, i18next.

---

## File structure

- `setting/system_setting/volc_asset.go`: dedicated callback-origin validation.
- `model/volc_asset_authorization.go`: hashed pending sessions and atomic completion.
- `relay/channel/task/doubao/asset_authorization_service.go`: H5 creation, polling, expiry, and binding.
- `controller/seedance_asset.go`: session-authenticated dashboard REST adapter.
- `web/src/services/seedanceAssets.js`: frontend API boundary.
- `web/src/components/seedance-assets/state.js`: pure state and polling rules.
- `web/src/components/seedance-assets/AuthorizationCard.jsx`: consent and QR/mobile handoff.
- `web/src/components/seedance-assets/AssetLibrary.jsx`: asset URL registration and management.
- `web/src/pages/SeedanceAssets/index.jsx`: orchestration and `sessionStorage` lifecycle.
- `web/src/pages/SeedanceAuthorizationCallback/index.jsx`: credential-free public return page.

### Task 1: Add the dedicated H5 callback origin

**Files:**
- Modify: `setting/system_setting/volc_asset.go`
- Modify: `setting/system_setting/volc_asset_test.go`
- Modify: `web/src/components/settings/VolcAssetSetting.jsx`

- [ ] **Step 1: Write the failing URL tests**

```go
func TestVolcAssetSettingsAuthorizationCallbackURL(t *testing.T) {
	tests := []struct {
		base string
		want string
		bad  bool
	}{
		{base: "https://www.token123.co", want: "https://www.token123.co/seedance/authorization/callback"},
		{base: "https://www.token123.co/", want: "https://www.token123.co/seedance/authorization/callback"},
		{base: "http://www.token123.co", bad: true},
		{base: "https://www.token123.co/path", bad: true},
		{base: "https://www.token123.co?next=bad", bad: true},
	}
	for _, tt := range tests {
		cfg := VolcAssetSettings{AuthorizationCallbackBaseURL: tt.base}
		got, err := cfg.GetAuthorizationCallbackURL()
		require.Equal(t, tt.bad, err != nil)
		require.Equal(t, tt.want, got)
	}
}
```

- [ ] **Step 2: Verify that the test fails**

Run `go test ./setting/system_setting -run AuthorizationCallback -count=1`.

Expected: compile failure for the missing field and method.

- [ ] **Step 3: Implement callback validation**

Add `AuthorizationCallbackBaseURL` to private/public settings and use:

```go
func (v VolcAssetSettings) GetAuthorizationCallbackURL() (string, error) {
	raw := strings.TrimSpace(v.AuthorizationCallbackBaseURL)
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" ||
		(parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" ||
		parsed.Fragment != "" || parsed.User != nil {
		return "", errors.New("a valid HTTPS Seedance authorization callback origin is required")
	}
	return strings.TrimRight(raw, "/") + "/seedance/authorization/callback", nil
}
```

- [ ] **Step 4: Add the root setting field**

Extend the existing form payload with `authorization_callback_base_url`. Render a required input labelled `t('真人授权回调站点')` and explain that production uses `https://www.token123.co`, not the API domain. Keep secret redaction and blank-secret merge unchanged.

- [ ] **Step 5: Verify and commit**

Run `go test ./setting/system_setting ./model ./controller -run 'VolcAsset|AuthorizationCallback' -count=1`.

```bash
git add setting/system_setting/volc_asset.go setting/system_setting/volc_asset_test.go web/src/components/settings/VolcAssetSetting.jsx
git commit -m "feat: configure Seedance authorization callback"
```

### Task 2: Persist hashed, user-bound sessions

**Files:**
- Create: `model/volc_asset_authorization.go`
- Create: `model/volc_asset_authorization_test.go`
- Modify: `model/volc_asset_group.go`
- Modify: `model/main.go`

- [ ] **Step 1: Write failing SQLite model tests**

Create tests named:

```go
func TestStartVolcAssetAuthorizationExpiresOlderPendingSession(t *testing.T)
func TestFindPendingVolcAssetAuthorizationRequiresSameUserAndDigest(t *testing.T)
func TestFindPendingVolcAssetAuthorizationRejectsExpiredSession(t *testing.T)
func TestCompleteVolcAssetAuthorizationBindsGroupAndConsumesSession(t *testing.T)
func TestCompleteVolcAssetAuthorizationIsIdempotentForSameBinding(t *testing.T)
```

Include the secrecy assertion:

```go
require.NotContains(t, stored.TokenHash, "raw-byted-token")
require.Len(t, stored.TokenHash, 64)
```

- [ ] **Step 2: Verify failure**

Run `go test ./model -run VolcAssetAuthorization -count=1`.

Expected: compile failure because the model functions do not exist.

- [ ] **Step 3: Implement the portable model**

```go
type VolcAssetAuthorizationSession struct {
	Id        int    `json:"id" gorm:"primaryKey"`
	UserId    int    `json:"user_id" gorm:"index;not null"`
	TokenHash string `json:"-" gorm:"type:varchar(64);uniqueIndex;not null"`
	Status    string `json:"status" gorm:"type:varchar(16);index;not null"`
	ExpiresAt int64  `json:"expires_at" gorm:"bigint;index;not null"`
	CreatedAt int64  `json:"created_at" gorm:"bigint;not null"`
	UpdatedAt int64  `json:"updated_at" gorm:"bigint;not null"`
}

const (
	VolcAssetAuthorizationPending   = "Pending"
	VolcAssetAuthorizationCompleted = "Completed"
	VolcAssetAuthorizationExpired   = "Expired"
	VolcAssetAuthorizationFailed    = "Failed"
)
```

Implement `StartVolcAssetAuthorization`, `FindPendingVolcAssetAuthorization`, `ExpireVolcAssetAuthorization`, and `CompleteVolcAssetAuthorization` with GORM only. Starting a session expires older pending rows for that user. Completion uses `DB.Transaction`, requires one matching pending digest, then saves the binding.

When the pending update affects zero rows, `CompleteVolcAssetAuthorization` queries the same `(user_id, token_hash)` once. If its status is already `Completed`, return success without another group write; every other status returns an authorization-session error. This is the only idempotent replay case.

- [ ] **Step 4: Make the group upsert transaction-aware**

```go
func SaveVolcAssetUserGroup(userId int, groupId string) error {
	return saveVolcAssetUserGroup(DB, userId, groupId)
}

func saveVolcAssetUserGroup(tx *gorm.DB, userId int, groupId string) error {
	if userId <= 0 || groupId == "" {
		return fmt.Errorf("invalid Volcengine asset user group binding")
	}
	now := common.GetTimestamp()
	var existing VolcAssetUserGroup
	err := tx.Where("user_id = ?", userId).First(&existing).Error
	if err == nil {
		return tx.Model(&existing).Updates(map[string]any{
			"group_id": groupId, "updated_at": now,
		}).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	created := VolcAssetUserGroup{
		UserId: userId, GroupId: groupId, CreatedAt: now, UpdatedAt: now,
	}
	if err := tx.Create(&created).Error; err != nil {
		var winner VolcAssetUserGroup
		if readErr := tx.Where("user_id = ?", userId).First(&winner).Error; readErr == nil {
			return tx.Model(&winner).Updates(map[string]any{
				"group_id": groupId, "updated_at": now,
			}).Error
		}
		return err
	}
	return nil
}
```

- [ ] **Step 5: Register both migration paths**

Add `&VolcAssetAuthorizationSession{}` next to `&VolcAssetUserGroup{}` in `migrateDB` and `migrateDBFast`. Do not add raw SQL or a database-specific type.

- [ ] **Step 6: Verify and commit**

Run `go test ./model -run 'VolcAsset(UserGroup|Authorization)' -count=1`.

```bash
git add model/volc_asset_authorization.go model/volc_asset_authorization_test.go model/volc_asset_group.go model/main.go
git commit -m "feat: bind Seedance authorization sessions to users"
```

### Task 3: Implement the H5 authorization service

**Files:**
- Create: `relay/channel/task/doubao/asset_authorization_service.go`
- Create: `relay/channel/task/doubao/asset_authorization_service_test.go`

- [ ] **Step 1: Write failing service tests**

Cover these contracts with fake API/repositories:

```go
func TestAuthorizationStatusDoesNotReturnGroupID(t *testing.T)
func TestCreateAuthorizationSessionUsesConfiguredCallbackAndStoresDigest(t *testing.T)
func TestCreateAuthorizationSessionRejectsMissingH5Data(t *testing.T)
func TestPollAuthorizationRejectsAnotherUsersDigest(t *testing.T)
func TestPollAuthorizationReturnsPendingForEmptyGroup(t *testing.T)
func TestPollAuthorizationExpiresObserved40004(t *testing.T)
func TestPollAuthorizationCompletesBinding(t *testing.T)
```

- [ ] **Step 2: Verify failure**

Run `go test ./relay/channel/task/doubao -run Authorization -count=1`.

Expected: compile failure for the missing service.

- [ ] **Step 3: Add dashboard-safe types**

```go
type AuthorizationStatus struct {
	Configured bool  `json:"configured"`
	Authorized bool  `json:"authorized"`
	Pending    bool  `json:"pending"`
	UpdatedAt  int64 `json:"updated_at,omitempty"`
}

type AuthorizationSession struct {
	BytedToken string `json:"byted_token"`
	H5Link     string `json:"h5_link"`
	ExpiresAt  int64  `json:"expires_at"`
}

type AuthorizationResult struct {
	Status     string `json:"status"`
	Authorized bool   `json:"authorized"`
}
```

No dashboard response type contains `GroupId`.

- [ ] **Step 4: Implement creation and polling**

Use an injectable clock and five-minute TTL. Hash only with the project helper:

```go
digest := hex.EncodeToString(common.Sha256Raw([]byte(strings.TrimSpace(rawToken))))
```

Creation uses `GetAuthorizationCallbackURL`, calls `CreateVisualValidateSession`, requires non-empty token/link, and stores only the digest. Polling checks the current user/digest before BytePlus, treats empty `GroupId` as pending, maps the observed upstream code `40004` to expired, and atomically completes the binding on success.

`AuthorizationStatus.Configured` is true only when Access Key, Secret Key, and the validated callback origin are all present. `Authorized` comes from the local user binding, while `Pending` comes from an unexpired session; status lookup never calls BytePlus.

- [ ] **Step 5: Preserve the developer API**

Do not change the response contract of `/doubao/open/GetVisualValidateResult`; its existing Token API continues returning `GroupId`. Digest ownership applies only to dashboard sessions created by this service.

- [ ] **Step 6: Verify and commit**

Run `go test ./relay/channel/task/doubao -run 'Authorization|AssetService' -count=1`.

```bash
git add relay/channel/task/doubao/asset_authorization_service.go relay/channel/task/doubao/asset_authorization_service_test.go
git commit -m "feat: add user-bound Seedance H5 authorization"
```

### Task 4: Expose the dashboard REST API

**Files:**
- Create: `controller/seedance_asset.go`
- Create: `controller/seedance_asset_test.go`
- Modify: `router/api-router.go`
- Create: `router/api_router_seedance_asset_test.go`

- [ ] **Step 1: Write failing controller/router tests**

```go
func TestSeedanceAuthorizationStatusRedactsGroupID(t *testing.T)
func TestSeedanceCreateAuthorizationSessionUsesNoClientCallback(t *testing.T)
func TestSeedancePollAuthorizationRequiresBytedToken(t *testing.T)
func TestSeedanceAssetCreateIgnoresClientGroupID(t *testing.T)
func TestSeedanceAssetNotOwnedReturns404(t *testing.T)
func TestSeedanceDashboardRoutesRequireUserAuth(t *testing.T)
```

Expected status envelope:

```json
{"success":true,"data":{"configured":true,"authorized":false,"pending":false}}
```

- [ ] **Step 2: Verify failure**

Run `go test ./controller ./router -run Seedance -count=1`.

- [ ] **Step 3: Implement a thin injectable controller**

The controller owns no BytePlus logic. It creates the production authorization/asset services, or accepts fakes in tests. Decode bodies with `common.DecodeJson`, take IDs from `c.Param("id")`, and return stable mappings:

```text
invalid_request=400
authorization_pending=200 data.status=pending
authorization_expired=410
asset_authorization_required=409
asset_not_found=404
asset_library_not_configured=503
other upstream error=502/503 with safe text
```

- [ ] **Step 4: Mount exact `UserAuth` routes**

```go
seedance := apiRouter.Group("/seedance/assets")
seedance.Use(middleware.UserAuth())
{
	seedance.GET("/authorization", controller.GetSeedanceAuthorizationStatus)
	seedance.POST("/authorization/session", controller.CreateSeedanceAuthorizationSession)
	seedance.POST("/authorization/result", controller.GetSeedanceAuthorizationResult)
	seedance.GET("/", controller.ListSeedanceAssets)
	seedance.POST("/", controller.CreateSeedanceAsset)
	seedance.GET("/:id", controller.GetSeedanceAsset)
	seedance.PATCH("/:id", controller.UpdateSeedanceAsset)
	seedance.DELETE("/:id", controller.DeleteSeedanceAsset)
}
```

- [ ] **Step 5: Verify and commit**

Run `go test ./controller ./router -run Seedance -count=1`.

```bash
git add controller/seedance_asset.go controller/seedance_asset_test.go router/api-router.go router/api_router_seedance_asset_test.go
git commit -m "feat: expose Seedance asset console API"
```

### Task 5: Add the frontend client and state machine

**Files:**
- Create: `web/src/services/seedanceAssets.js`
- Create: `web/src/components/seedance-assets/state.js`
- Create: `web/src/components/seedance-assets/state.test.js`

- [ ] **Step 1: Write the failing Bun state test**

```js
import { describe, expect, test } from 'bun:test';
import {
  AUTH_STATE,
  authorizationStateFromStatus,
  shouldContinueAuthorizationPolling,
} from './state';

describe('Seedance authorization state', () => {
  test('configuration and authorization have deterministic priority', () => {
    expect(authorizationStateFromStatus({ configured: false })).toBe(
      AUTH_STATE.UNCONFIGURED,
    );
    expect(
      authorizationStateFromStatus({ configured: true, authorized: true }),
    ).toBe(AUTH_STATE.AUTHORIZED);
  });
  test('polling ends at a terminal state or five minutes', () => {
    expect(shouldContinueAuthorizationPolling(AUTH_STATE.WAITING, 299999)).toBe(true);
    expect(shouldContinueAuthorizationPolling(AUTH_STATE.WAITING, 300000)).toBe(false);
    expect(shouldContinueAuthorizationPolling(AUTH_STATE.AUTHORIZED, 1000)).toBe(false);
  });
});
```

- [ ] **Step 2: Verify failure**

Run `cd web && bun test src/components/seedance-assets/state.test.js`.

- [ ] **Step 3: Implement pure states and one API surface**

Export only `checking`, `unconfigured`, `unauthorized`, `creating`, `waiting`, `authorized`, `expired`, and `failed`. Keep timers/browser APIs outside `state.js`.

```js
export const seedanceAssetsApi = {
  authorizationStatus: () => API.get('/api/seedance/assets/authorization'),
  createAuthorizationSession: () =>
    API.post('/api/seedance/assets/authorization/session'),
  authorizationResult: (bytedToken) =>
    API.post('/api/seedance/assets/authorization/result', {
      byted_token: bytedToken,
    }),
  list: (params) => API.get('/api/seedance/assets/', { params }),
  create: (payload) => API.post('/api/seedance/assets/', payload),
  get: (id) => API.get(`/api/seedance/assets/${encodeURIComponent(id)}`),
  update: (id, name) => API.patch(`/api/seedance/assets/${encodeURIComponent(id)}`, { name }),
  remove: (id) => API.delete(`/api/seedance/assets/${encodeURIComponent(id)}`),
};
```

- [ ] **Step 4: Verify and commit**

Run `cd web && bun test src/components/seedance-assets/state.test.js`.

```bash
git add web/src/services/seedanceAssets.js web/src/components/seedance-assets/state.js web/src/components/seedance-assets/state.test.js
git commit -m "feat: add Seedance console client state"
```

### Task 6: Build authorization and asset-management UI

**Files:**
- Create: `web/src/components/seedance-assets/AuthorizationCard.jsx`
- Create: `web/src/components/seedance-assets/AssetLibrary.jsx`
- Create: `web/src/pages/SeedanceAssets/index.jsx`
- Create: `web/src/pages/SeedanceAuthorizationCallback/index.jsx`
- Modify: `web/src/App.jsx`
- Modify: `web/src/components/layout/SiderBar.jsx`

- [ ] **Step 1: Implement a controlled authorization card**

```jsx
<AuthorizationCard
  state={authorizationState}
  consented={consented}
  onConsentChange={setConsented}
  h5Link={h5Link}
  expiresAt={expiresAt}
  errorMessage={errorMessage}
  onStart={startAuthorization}
  onRetry={startAuthorization}
/>
```

Use `QRCodeSVG` on desktop, a direct same-tab H5 action on mobile, and always provide a text/copy link. Disable start until consent is checked. Never auto-open a popup.

Use semantic `Button`, `Checkbox`, labelled inputs, and visible focus states. Every colored status also includes an icon and text so keyboard and non-color users receive the same information.

- [ ] **Step 2: Implement the controlled asset library**

Create sends only `{url, asset_type}`. Show text plus icon for `Active`, `Processing`, and `Failed`. Only `Active` may execute:

```js
navigator.clipboard.writeText(`asset://${asset.id}`);
```

Rename uses an explicit modal. Delete uses `Modal.confirm`. Never render `group_id`.

- [ ] **Step 3: Implement page orchestration**

Use only `sessionStorage['seedance_authorization_token']`. Load status on mount, resume polling when a token exists, poll every three seconds for at most five minutes, clear on terminal states, and pause asset polling while the document is hidden. Never put the token in a URL, log, analytics event, or error message.

- [ ] **Step 4: Implement the public callback**

Render safe completion guidance and a link to `/console/seedance-assets`. Do not parse or reflect query parameters.

- [ ] **Step 5: Add routes and sidebar item**

Add private `/console/seedance-assets`, public `/seedance/authorization/callback`, `seedanceAssets` in `routerMap`, and `真人素材库` in workspace navigation.

- [ ] **Step 6: Format, build, and commit**

Run `cd web && bunx prettier --write src/services/seedanceAssets.js src/components/seedance-assets src/pages/SeedanceAssets src/pages/SeedanceAuthorizationCallback src/App.jsx src/components/layout/SiderBar.jsx && bun run build`.

```bash
git add web/src/services/seedanceAssets.js web/src/components/seedance-assets web/src/pages/SeedanceAssets web/src/pages/SeedanceAuthorizationCallback web/src/App.jsx web/src/components/layout/SiderBar.jsx
git commit -m "feat: add Seedance real-person asset console"
```

### Task 7: Add translations and update documentation

**Files:**
- Modify: `web/src/i18n/locales/{zh,en,fr,ru,ja,vi}.json`
- Modify: `docs/seedance-asset-api.md`

- [ ] **Step 1: Add approved legal copy**

Required Chinese source meanings include:

```text
真人素材库
本人活体核验与肖像授权
我确认由本人完成活体核验，并授权平台为 Seedance 真人素材生成处理我的人脸与肖像信息。
请使用有摄像头的手机扫码完成授权；原设备会自动等待结果。
授权完成，请返回原设备。
授权链接已过期，请重新生成。
重新授权会替换当前真人绑定，原素材不会自动迁移。
若所有可用设备都没有摄像头，将无法完成真人授权。
```

English uses “liveness verification” and “portrait authorization”, never “face recognition login”. Use these exact translations for the two core legal headings and consent sentence; the shorter operational strings use the same terminology and are generated by `i18n:sync` before manual review:

| Locale | Library title | Authorization heading | Consent sentence |
| --- | --- | --- | --- |
| `en` | Real-Person Asset Library | Liveness Verification and Portrait Authorization | I confirm that I will complete the liveness check myself and authorize the platform to process my face and portrait information for Seedance real-person asset generation. |
| `fr` | Bibliothèque de ressources de personne réelle | Vérification de présence et autorisation du droit à l’image | Je confirme que j’effectuerai moi-même la vérification de présence et j’autorise la plateforme à traiter les informations relatives à mon visage et à mon image pour la génération de ressources Seedance représentant une personne réelle. |
| `ru` | Библиотека материалов с реальным человеком | Проверка живого присутствия и разрешение на использование изображения | Я подтверждаю, что лично пройду проверку живого присутствия, и разрешаю платформе обрабатывать данные моего лица и изображения для создания материалов Seedance с реальным человеком. |
| `ja` | 実在人物素材ライブラリ | 生体確認と肖像利用の許諾 | 私本人が生体確認を完了し、Seedance の実在人物素材生成のために、プラットフォームが私の顔および肖像情報を処理することに同意します。 |
| `vi` | Thư viện tư liệu người thật | Xác minh người thật và ủy quyền sử dụng chân dung | Tôi xác nhận sẽ tự mình hoàn tất bước xác minh người thật và cho phép nền tảng xử lý thông tin khuôn mặt, chân dung của tôi để tạo tư liệu người thật bằng Seedance. |

After synchronization, inspect every new Seedance key in all six locale files and fail the task if any non-`zh` value is still identical to the Chinese source.

- [ ] **Step 2: Synchronize and lint locales**

Run `cd web && bun run i18n:extract && bun run i18n:sync && bun run i18n:lint`.

Expected: no missing or unused Seedance keys.

- [ ] **Step 3: Update user/API documentation**

Document dashboard session routes, QR/mobile handoff, five-minute digest-only session, dedicated callback origin, URL-only asset registration, developer Token API compatibility, and the non-bypassable liveness/consent requirement.

- [ ] **Step 4: Build and commit**

Run `cd web && bun run build`.

```bash
git add web/src/i18n/locales docs/seedance-asset-api.md
git commit -m "docs: explain Seedance user authorization"
```

### Task 8: Regression, release, and production validation

**Files:**
- Modify only files required by reproduced failures.

- [ ] **Step 1: Run focused and full tests**

```bash
go test ./setting/system_setting ./model ./relay/channel/task/doubao ./controller ./router -run 'VolcAsset|Seedance|Authorization' -count=1
go test ./... -count=1
cd web && bun test src/components/seedance-assets/state.test.js && bun run i18n:lint && bun run build
```

Expected: PASS. A failure may be called unrelated only after reproducing it on `origin/main`.

- [ ] **Step 2: Run local browser QA**

Verify anonymous 401, unconfigured state, desktop QR, 375px mobile action, expired regeneration, active-only `asset://` copy, delete confirmation, and query-reflection resistance. Browser console, URL, and network logs must not expose `BytedToken`.

- [ ] **Step 3: Review, push, and open the PR**

Run `git diff --check`, confirm no secret material, then:

```bash
git push -u origin agent/seedance-user-authorization
gh pr create --base main --head agent/seedance-user-authorization --title "feat: add Seedance real-person asset console" --body-file /tmp/seedance-user-auth-pr.md
```

- [ ] **Step 4: Merge and deploy after CI/review**

Use the repository’s safe merge path. Wait for the deploy workflow, then set only `VolcAssetConfig.AuthorizationCallbackBaseURL=https://www.token123.co`; preserve AK/SK, region, project, and `LivenessFace`. Verify secret redaction.

- [ ] **Step 5: Run production auth smoke**

Anonymous dashboard status returns 401, administrator status returns 200, and existing `/doubao/open/*` Token API behavior remains unchanged.

- [ ] **Step 6: Complete the required human H5 step**

From `qw52789@gmail.com`, create one session. The administrator scans with a camera phone, reviews terms, and personally completes liveness. Verify desktop auto-transition, a 64-character stored digest, and exactly one active user binding.

- [ ] **Step 7: Validate asset CRUD and isolation**

Register an administrator-owned authorized media URL, poll to `Active`, copy `asset://<id>`, rename, and verify cross-user access is 404 through local/transaction-level tests. Do not log in, fund, or modify any `test20xxx` production account.

- [ ] **Step 8: Gate costed generation on pricing correction**

Do not submit a costed production video while the known `ModelRatio`/`USDExchangeRate=6.7` double-conversion remains. After the separate price fix is explicitly approved and verified, use an administrator-owned temporary API Token for one minimum-cost 480p/5-second Fast request with `asset://<id>`, poll to success, verify final usage reconciliation, and delete the temporary Token and test asset.

- [ ] **Step 9: Canary and release record**

Observe authorization create/result errors, asset CRUD 5xx, frontend errors, ownership 404 behavior, and final task settlement. Roll back the app on auth/privacy/compatibility regression. Record deployed commit and evidence without raw tokens, group IDs, keys, or portrait URLs.
