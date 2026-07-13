# Seedance Multi-Actor Authorization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let one Token123 account manage multiple independently authorized Seedance actors with selectable consent periods, immediate expiry/revocation enforcement, safe mobile handoff, and isolated asset libraries.

**Architecture:** Introduce a first-class `VolcAssetActor` owned by a user and scope authorization sessions and assets to it. Keep legacy user-group routes mapped to an idempotently migrated default actor, while the dashboard uses actor-specific routes. Enforce authorization synchronously in the asset service and in the Doubao task adaptor before any upstream request.

**Tech Stack:** Go 1.22, Gin, GORM v2, SQLite/MySQL/PostgreSQL-compatible models, React 18, Semi UI, i18next, Bun.

---

## File map

- Create `model/volc_asset_actor.go`: actor states, duration validation, expiry calculation, ownership queries, revoke, default-actor migration, and asset-to-actor mapping.
- Create `model/volc_asset_actor_test.go`: cross-user isolation, duration boundaries, expiry/revocation, and idempotent migration tests.
- Modify `model/volc_asset_authorization.go`: scope sessions by actor and persist consent/invitation/revocation digests.
- Modify `model/volc_asset_authorization_test.go`: multi-actor session isolation and consent tests.
- Modify `model/main.go`: AutoMigrate the new portable models and run idempotent legacy binding migration.
- Create `relay/channel/task/doubao/actor_service.go`: actor CRUD and authorization guard shared by HTTP and video relay paths.
- Modify `relay/channel/task/doubao/asset_authorization_service.go`: actor-scoped sessions, opaque public handoff, consent, completion, and revocation.
- Modify `relay/channel/task/doubao/asset_service.go`: explicit actor-scoped CRUD plus legacy default-actor compatibility.
- Modify `relay/channel/task/doubao/adaptor.go`: reject expired/revoked `asset://` references before calling BytePlus.
- Create `controller/seedance_actor.go`: actor, public consent, revoke, and actor asset endpoints.
- Modify `controller/seedance_asset.go`: shared safe error codes and default-actor compatibility.
- Modify `router/api-router.go`: authenticated actor routes and rate-limited public consent/revoke routes.
- Modify `router/api_router_seedance_asset_test.go`: exact route/auth coverage.
- Modify `web/src/services/seedanceAssets.js`: actor and public handoff client methods.
- Create `web/src/components/seedance-assets/ActorManager.jsx`: actor selector, creation/rename/revoke controls, and duration picker.
- Modify `web/src/components/seedance-assets/AuthorizationCard.jsx`: show selected actor, period, and Token123 handoff QR.
- Modify `web/src/components/seedance-assets/AssetLibrary.jsx`: actor-scoped calls and expired read/delete behavior.
- Modify `web/src/components/seedance-assets/state.js`: actor statuses, expiry warning calculations, and tests.
- Modify `web/src/pages/SeedanceAssets/index.jsx`: multi-actor orchestration and per-actor polling.
- Create `web/src/pages/SeedanceAuthorizationConsent/index.jsx`: public mobile consent/handoff page.
- Modify `web/src/pages/SeedanceAuthorizationCallback/index.jsx`: display revocation receipt after H5 callback.
- Modify `web/src/App.jsx`: register the public consent route.
- Modify `web/src/i18n/locales/{zh-CN,zh-TW,en,fr,ru,ja,vi}.json`: all new UI strings.

### Task 1: Actor model, expiry rules, and legacy migration

**Files:**
- Create: `model/volc_asset_actor.go`
- Create: `model/volc_asset_actor_test.go`
- Modify: `model/main.go`

- [ ] **Step 1: Write failing actor lifecycle tests**

Add table-driven tests that assert:

```go
for _, days := range []int{30, 60, 180, 365} {
    actor, err := CreateVolcAssetActor(42, "演员", &days, 100)
    require.NoError(t, err)
    require.Equal(t, days, *actor.AuthorizationDurationDays)
}
invalidDays := 31
_, err := CreateVolcAssetActor(42, "演员", &invalidDays, 100)
require.ErrorIs(t, err, ErrVolcAssetInvalidDuration)
```

Also assert that two actors under user 42 coexist, user 43 receives `ErrVolcAssetActorNotFound`, `now == expires_at` returns `ErrVolcAssetActorExpired`, revoked actors return `ErrVolcAssetActorRevoked`, and running legacy migration twice produces one default actor per old binding.

- [ ] **Step 2: Verify RED**

Run: `go test ./model -run 'TestVolcAssetActor|TestMigrateVolcAsset' -count=1`

Expected: FAIL because `VolcAssetActor` APIs do not exist.

- [ ] **Step 3: Implement the portable model**

Define only cross-database GORM types:

```go
type VolcAssetActor struct {
    Id int `gorm:"primaryKey"`
    UserId int `gorm:"index;not null"`
    DisplayName string `gorm:"type:varchar(128);not null"`
    GroupId string `gorm:"type:text" json:"-"`
    Status string `gorm:"type:varchar(16);index;not null"`
    AuthorizationDurationDays *int
    AuthorizedAt *int64 `gorm:"type:bigint"`
    AuthorizationExpiresAt *int64 `gorm:"type:bigint;index"`
    ConsentVersion string `gorm:"type:varchar(32)"`
    IsDefault bool `gorm:"index;not null"`
    CreatedAt int64 `gorm:"type:bigint;not null"`
    UpdatedAt int64 `gorm:"type:bigint;not null"`
}

type VolcAssetActorAsset struct {
    Id int `gorm:"primaryKey"`
    UserId int `gorm:"index;not null"`
    ActorId int `gorm:"uniqueIndex:idx_actor_asset;not null"`
    AssetId string `gorm:"type:varchar(191);uniqueIndex:idx_actor_asset;index;not null"`
    CreatedAt int64 `gorm:"type:bigint;not null"`
    UpdatedAt int64 `gorm:"type:bigint;not null"`
}
```

Implement `CreateVolcAssetActor`, `ListVolcAssetActors`, `GetVolcAssetActorForUser`, `RenameVolcAssetActor`, `ActivateVolcAssetActor`, `RequireVolcAssetActorActiveAt`, `RevokeVolcAssetActor`, `SaveVolcAssetActorAsset`, `FindVolcAssetActorByAsset`, and `MigrateVolcAssetUserGroupsToActors`. Use GORM transactions and `clause.OnConflict`; do not use database-specific SQL.

The active guard must compare Unix seconds synchronously:

```go
if actor.Status == VolcAssetActorRevoked { return ErrVolcAssetActorRevoked }
if actor.AuthorizationExpiresAt != nil && *actor.AuthorizationExpiresAt <= now {
    // best-effort materialize Expired, then always reject this request
    return ErrVolcAssetActorExpired
}
if actor.Status != VolcAssetActorActive || strings.TrimSpace(actor.GroupId) == "" {
    return ErrVolcAssetActorAuthorizationRequired
}
```

- [ ] **Step 4: Register migrations and verify GREEN**

Add `VolcAssetActor` and `VolcAssetActorAsset` to both normal and test migration lists in `model/main.go`; invoke the idempotent legacy migration after AutoMigrate.

Run: `go test ./model -run 'TestVolcAssetActor|TestMigrateVolcAsset' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add model/volc_asset_actor.go model/volc_asset_actor_test.go model/main.go
git commit -m "feat: model Seedance actors and authorization expiry"
```

### Task 2: Actor-scoped sessions, mobile consent, and revocation

**Files:**
- Modify: `model/volc_asset_authorization.go`
- Modify: `model/volc_asset_authorization_test.go`
- Create: `model/volc_asset_authorization_event.go`
- Modify: `model/main.go`
- Modify: `relay/channel/task/doubao/asset_authorization_service.go`
- Modify: `relay/channel/task/doubao/asset_authorization_service_test.go`

- [ ] **Step 1: Write failing isolation and consent tests**

Test that sessions for actor A and B can both remain pending; a new session expires only an older pending session for the same actor; `FindPending` requires `user_id + actor_id + token_hash`; public details reveal display name/duration but never GroupId; consent is one-way/idempotent; completion before consent is rejected; completion activates only the matching actor; public revocation changes only that actor.

Use fixed `now` and require exact expiry:

```go
require.Equal(t, int64(100+30*86400), *actor.AuthorizationExpiresAt)
require.Nil(t, longTermActor.AuthorizationExpiresAt)
```

- [ ] **Step 2: Verify RED**

Run: `go test ./model ./relay/channel/task/doubao -run 'Test.*(ActorAuthorization|Consent|Revocation|MultiActor)' -count=1`

Expected: FAIL on missing actor/session signatures.

- [ ] **Step 3: Extend the session and event models**

Add `ActorId`, nullable `DurationDays`, `ConsentVersion`, `InvitationTokenHash`, `RevocationTokenHash`, and nullable `ConsentAcceptedAt`. Keep `TokenHash` unique and store only SHA-256 hex digests. Define append-only events with event type, user, actor, duration snapshot, consent version, and timestamp; do not add a generic update method. Register the extended session and event models in both `AutoMigrate` lists in `model/main.go`.

Change repository signatures consistently:

```go
Start(userID, actorID int, tokenHash, invitationHash, revocationHash string, durationDays *int, consentVersion string, expiresAt, now int64) error
FindPending(userID, actorID int, tokenHash string, now int64) error
AcceptConsent(invitationHash string, now int64) (*VolcAssetAuthorizationSession, error)
Complete(userID, actorID int, tokenHash, groupID string, now int64) error
RevokeByToken(revocationHash string, now int64) error
```

- [ ] **Step 4: Implement the handoff service**

Generate invitation and revocation tokens with `common.GenerateRandomKey(48)`. Return to the authenticated browser:

```go
type AuthorizationSession struct {
    BytedToken string `json:"byted_token"`
    H5Link string `json:"h5_link"`
    InvitationToken string `json:"invitation_token"`
    ExpiresAt int64 `json:"expires_at"`
}
```

The dashboard builds a public URL fragment client-side. Public details/consent accept only the invitation token in JSON bodies. Consent returns the revocation token stored transiently in the phone session; completion requires `ConsentAcceptedAt != nil` and activates the actor using the session duration snapshot.

- [ ] **Step 5: Verify GREEN and commit**

Run: `go test ./model ./relay/channel/task/doubao -run 'Test.*(ActorAuthorization|Consent|Revocation|MultiActor)' -count=1`

Expected: PASS.

```bash
git add model/volc_asset_authorization.go model/volc_asset_authorization_test.go model/volc_asset_authorization_event.go model/main.go relay/channel/task/doubao/asset_authorization_service.go relay/channel/task/doubao/asset_authorization_service_test.go
git commit -m "feat: scope Seedance consent sessions to actors"
```

### Task 3: Actor and public HTTP APIs

**Files:**
- Create: `controller/seedance_actor.go`
- Create: `controller/seedance_actor_test.go`
- Create: `relay/channel/task/doubao/actor_service.go`
- Modify: `controller/seedance_asset.go`
- Modify: `router/api-router.go`
- Modify: `router/api_router_seedance_asset_test.go`

- [ ] **Step 1: Write failing route and response tests**

Require all actor routes from the design, verify authenticated routes return 401 without a session, and verify the three public POST routes exist without `UserAuth`. Controller tests must assert `GroupId`, token hashes, and raw BytedToken are absent from status/list JSON. Cross-user actors must produce 404 `actor_not_found`; expired and revoked actors must produce the exact 409 codes.

- [ ] **Step 2: Verify RED**

Run: `go test ./controller ./router -run 'TestSeedance(Actor|Public|Dashboard)' -count=1`

Expected: FAIL due to missing routes/controllers.

- [ ] **Step 3: Implement thin controllers and routing**

Implement `ActorService` as the narrow owner of list/create/rename/delete/revoke operations and safe actor DTOs. Parse actor IDs with `strconv.Atoi`, reject non-positive IDs, use `common.DecodeJson`, and return the established envelope:

```json
{"success":true,"data":{}}
```

Register public endpoints before the authenticated actor group and protect them with `middleware.CriticalRateLimit()`. Never use query/path tokens. Map model/service errors to the stable codes from the design and mask all other upstream details.

- [ ] **Step 4: Verify GREEN and commit**

Run: `go test ./controller ./router -run 'TestSeedance(Actor|Public|Dashboard)' -count=1`

Expected: PASS.

```bash
git add relay/channel/task/doubao/actor_service.go controller/seedance_actor.go controller/seedance_actor_test.go controller/seedance_asset.go router/api-router.go router/api_router_seedance_asset_test.go
git commit -m "feat: expose Seedance actor authorization APIs"
```

### Task 4: Actor-scoped asset CRUD and video-generation guard

**Files:**
- Modify: `relay/channel/task/doubao/asset_service.go`
- Modify: `relay/channel/task/doubao/asset_service_test.go`
- Modify: `relay/channel/task/doubao/adaptor.go`
- Modify: `relay/channel/task/doubao/adaptor_test.go`
- Modify: `controller/seedance_actor.go`

- [ ] **Step 1: Write failing asset and relay tests**

Cover two actors with different groups and multiple assets, cross-actor 404, expired actor read/rename/delete allowed, expired actor create rejected, and active asset mappings saved after create/list. In adaptor tests put `asset://asset-a` in image and video content and assert expired/revoked mappings return a task error before `BuildRequestBody`/upstream execution; another active actor remains accepted.

- [ ] **Step 2: Verify RED**

Run: `go test ./relay/channel/task/doubao -run 'Test.*(ActorAsset|AssetAuthorizationGuard)' -count=1`

Expected: FAIL because asset methods are user-only and adaptor has no guard.

- [ ] **Step 3: Add explicit actor methods**

Add `ListActorAssets`, `CreateActorAsset`, `GetActorAsset`, `UpdateActorAsset`, and `DeleteActorAsset`. Resolve the actor with `GetVolcAssetActorForUser`; call the active guard for create and the ownership/group guard for every operation. Save mappings for every returned list item and create response. Keep old methods as wrappers around the default actor.

- [ ] **Step 4: Guard `asset://` at validation time**

Extract all `asset://` IDs from request metadata content, deduplicate them, resolve each mapping for `c.GetInt("id")`, and call `RequireVolcAssetActorActiveAt(time.Now().Unix())`. Translate expiry/revocation/required errors to 409 task errors with `actor_authorization_expired`, `actor_authorization_revoked`, or `actor_authorization_required`.

- [ ] **Step 5: Verify GREEN and commit**

Run: `go test ./relay/channel/task/doubao -run 'Test.*(ActorAsset|AssetAuthorizationGuard)' -count=1`

Expected: PASS.

```bash
git add relay/channel/task/doubao/asset_service.go relay/channel/task/doubao/asset_service_test.go relay/channel/task/doubao/adaptor.go relay/channel/task/doubao/adaptor_test.go controller/seedance_actor.go
git commit -m "feat: isolate Seedance assets and enforce consent expiry"
```

### Task 5: Multi-actor dashboard and mobile consent handoff

**Files:**
- Modify: `web/src/services/seedanceAssets.js`
- Create: `web/src/components/seedance-assets/ActorManager.jsx`
- Modify: `web/src/components/seedance-assets/AuthorizationCard.jsx`
- Modify: `web/src/components/seedance-assets/AssetLibrary.jsx`
- Modify: `web/src/components/seedance-assets/state.js`
- Modify: `web/src/components/seedance-assets/state.test.js`
- Modify: `web/src/pages/SeedanceAssets/index.jsx`
- Create: `web/src/pages/SeedanceAuthorizationConsent/index.jsx`
- Create: `web/src/pages/SeedanceAuthorizationConsent/index.test.mjs`
- Modify: `web/src/pages/SeedanceAuthorizationCallback/index.jsx`
- Modify: `web/src/App.jsx`

- [ ] **Step 1: Write failing state and source-regression tests**

Assert duration options are exactly `[30, 60, 180, 365, null]`, default is 30, `expiresAt <= now` becomes expired, 7/3/1-day warning buckets are stable, actor selection scopes every asset API URL, and consent source reads secrets from `window.location.hash`, clears the fragment with `history.replaceState`, posts the invitation in a JSON body, and only redirects after consent succeeds.

- [ ] **Step 2: Verify RED**

Run: `cd web && bun test src/components/seedance-assets/state.test.js src/pages/SeedanceAuthorizationConsent/index.test.mjs`

Expected: FAIL because actor duration and consent page behavior are absent.

- [ ] **Step 3: Implement actor-aware service and state**

Use encoded actor IDs in every path. Export immutable duration options with Chinese translation keys and `DEFAULT_AUTHORIZATION_DURATION_DAYS = 30`. Return derived `expired`, `revoked`, and `expiring` status from server timestamps; never use browser time to authorize an action.

- [ ] **Step 4: Implement the dashboard**

`ActorManager` lists actors, creates one with display name/duration, selects it, and offers rename/revoke. `SeedanceAssets` stores polling state per selected actor. Build the QR target as:

```js
const fragment = new URLSearchParams({
  invitation: session.invitation_token,
  h5: session.h5_link,
});
const handoffUrl = `${window.location.origin}/seedance/authorization/consent#${fragment}`;
```

Do not render the raw H5 URL on the company page. `AssetLibrary` receives `actor`, disables create/copy when not active, but keeps list/rename/delete for expired or revoked actors.

- [ ] **Step 5: Implement the public phone pages**

The consent page immediately removes the fragment from the address bar after reading it, loads safe details via public POST, requires the actor checkbox, stores the returned revocation token in `sessionStorage`, then calls `window.location.assign(h5Link)`. The callback page reads that local token and presents a saveable Token123 revoke link without requiring login.

- [ ] **Step 6: Verify GREEN and commit**

Run: `cd web && bun test src/components/seedance-assets/state.test.js src/pages/SeedanceAuthorizationConsent/index.test.mjs`

Expected: PASS.

```bash
git add web/src/services/seedanceAssets.js web/src/components/seedance-assets web/src/pages/SeedanceAssets web/src/pages/SeedanceAuthorizationConsent web/src/pages/SeedanceAuthorizationCallback web/src/App.jsx
git commit -m "feat: add multi-actor Seedance asset dashboard"
```

### Task 6: Internationalization, compatibility, and full verification

**Files:**
- Modify: `web/src/i18n/locales/zh-CN.json`
- Modify: `web/src/i18n/locales/zh-TW.json`
- Modify: `web/src/i18n/locales/en.json`
- Modify: `web/src/i18n/locales/fr.json`
- Modify: `web/src/i18n/locales/ru.json`
- Modify: `web/src/i18n/locales/ja.json`
- Modify: `web/src/i18n/locales/vi.json`
- Modify tests only if a genuine defect is discovered during verification.

- [ ] **Step 1: Synchronize translations**

Add every new Chinese source key to all seven locale files. English must explicitly render “Long-term (the actor may revoke at any time)” rather than “permanent/irrevocable”. Other locales must preserve the same revocable meaning.

- [ ] **Step 2: Run formatting and focused tests**

Run:

```bash
gofmt -w model/volc_asset_actor.go model/volc_asset_actor_test.go model/volc_asset_authorization.go model/volc_asset_authorization_test.go model/volc_asset_authorization_event.go relay/channel/task/doubao/actor_service.go relay/channel/task/doubao/asset_authorization_service.go relay/channel/task/doubao/asset_authorization_service_test.go relay/channel/task/doubao/asset_service.go relay/channel/task/doubao/asset_service_test.go relay/channel/task/doubao/adaptor.go relay/channel/task/doubao/adaptor_test.go controller/seedance_actor.go controller/seedance_actor_test.go controller/seedance_asset.go router/api-router.go router/api_router_seedance_asset_test.go
go test ./model ./relay/channel/task/doubao ./controller ./router -count=1
```

Expected: all focused Go packages PASS.

- [ ] **Step 3: Run cross-project checks**

Run:

```bash
go test ./... -count=1
cd web && bun run i18n:lint
cd web && bun run build
git diff --check
```

Expected: Go exit 0, i18n lint exit 0, Vite build exit 0, and no whitespace errors.

- [ ] **Step 4: Review security invariants**

Search generated code and responses:

```bash
rg -n 'GroupId|BytedToken|InvitationToken|RevocationToken' controller web/src/pages/SeedanceAssets web/src/components/seedance-assets
rg -n 'encoding/json' model/volc_asset_actor.go model/volc_asset_authorization.go controller/seedance_actor.go
```

Expected: no GroupId or digest fields in public DTOs; raw tokens only appear in purpose-built short-lived request/response DTOs; no direct `encoding/json` marshal/unmarshal use.

- [ ] **Step 5: Commit the verified integration**

```bash
git add web/src/i18n/locales
git commit -m "feat: localize Seedance actor authorization"
git status --short
```

Expected: clean worktree after the final commit.

## Production handoff

Automated deployment can migrate schema and smoke-test authenticated/public routes. Real BytePlus H5 liveness must still be performed by the actor on a camera-equipped device; it is the only intentionally manual acceptance step. Production tests must use the administrator account `qw52789@gmail.com`, must not alter normal test-user balances, and must keep Seedance resale pricing at cost multiplied by `1.2`.
