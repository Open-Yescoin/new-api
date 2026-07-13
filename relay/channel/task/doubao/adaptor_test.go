package doubao

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupSeedanceAssetGuardTest(t *testing.T) {
	t.Helper()
	oldDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.VolcAssetActor{}, &model.VolcAssetActorAsset{}))
	model.DB = db
	t.Cleanup(func() { model.DB = oldDB })
}

func seedanceAssetReferenceRequest(assetID string) relaycommon.TaskSubmitReq {
	return relaycommon.TaskSubmitReq{Metadata: map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{"type": "image_url", "image_url": map[string]interface{}{"url": "asset://" + assetID}},
		},
	}}
}

func TestValidateSeedanceAssetReferencesEnforcesActorExpiryAndRevocation(t *testing.T) {
	setupSeedanceAssetGuardTest(t)
	days := 30
	expired, err := model.CreateVolcAssetActor(42, "已过期", &days, 10)
	require.NoError(t, err)
	require.NoError(t, model.ActivateVolcAssetActor(42, expired.Id, "group-expired", "v1", 100-30*86400))
	require.NoError(t, model.SaveVolcAssetActorAsset(42, expired.Id, "asset-expired", 20))

	revoked, err := model.CreateVolcAssetActor(42, "已撤回", nil, 11)
	require.NoError(t, err)
	require.NoError(t, model.ActivateVolcAssetActor(42, revoked.Id, "group-revoked", "v1", 20))
	require.NoError(t, model.SaveVolcAssetActorAsset(42, revoked.Id, "asset-revoked", 20))
	require.NoError(t, model.RevokeVolcAssetActor(42, revoked.Id, 30))

	err = validateSeedanceAssetReferences(42, seedanceAssetReferenceRequest("asset-expired"), 100)
	require.True(t, errors.Is(err, model.ErrVolcAssetActorExpired))
	err = validateSeedanceAssetReferences(42, seedanceAssetReferenceRequest("asset-revoked"), 100)
	require.True(t, errors.Is(err, model.ErrVolcAssetActorRevoked))
}

func TestValidateSeedanceAssetReferencesAllowsActiveActorOnlyForOwner(t *testing.T) {
	setupSeedanceAssetGuardTest(t)
	actor, err := model.CreateVolcAssetActor(42, "有效演员", nil, 10)
	require.NoError(t, err)
	require.NoError(t, model.ActivateVolcAssetActor(42, actor.Id, "group-active", "v1", 20))
	require.NoError(t, model.SaveVolcAssetActorAsset(42, actor.Id, "asset-active", 20))

	require.NoError(t, validateSeedanceAssetReferences(42, seedanceAssetReferenceRequest("asset-active"), 100))
	err = validateSeedanceAssetReferences(43, seedanceAssetReferenceRequest("asset-active"), 100)
	require.ErrorIs(t, err, model.ErrVolcAssetActorNotFound)
}

func TestForceSeedanceAliasResolution(t *testing.T) {
	req := relaycommon.TaskSubmitReq{Model: "Seedance-2.0-1080P-海外版"}
	if err := forceSeedanceAliasResolution(&req); err != nil {
		t.Fatalf("forceSeedanceAliasResolution returned error: %v", err)
	}
	if got := req.Metadata["resolution"]; got != "1080p" {
		t.Fatalf("resolution = %v, want 1080p", got)
	}
}

func TestForceSeedanceAliasResolutionRejectsConflict(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Model:    "Seedance-2.0-720P-海外版",
		Metadata: map[string]interface{}{"resolution": "4K"},
	}
	err := forceSeedanceAliasResolution(&req)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !strings.Contains(err.Error(), "requires resolution=720p") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestForceSeedance480PRejects4KConflict(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Model:    "Seedance-2.0-480P-海外版",
		Metadata: map[string]interface{}{"resolution": "4K"},
	}
	err := forceSeedanceAliasResolution(&req)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !strings.Contains(err.Error(), "requires resolution=480p") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSeedanceFastAllows720P(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Model:    "Seedance-2.0-fast-海外版",
		Metadata: map[string]interface{}{"resolution": "720P"},
	}
	if err := validateSeedanceModelResolution(&req); err != nil {
		t.Fatalf("validateSeedanceModelResolution returned error: %v", err)
	}
}

func TestValidateSeedanceFastRejects1080P(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Model:    "Seedance-2.0-fast-海外版",
		Metadata: map[string]interface{}{"resolution": "1080P"},
	}
	err := validateSeedanceModelResolution(&req)
	if err == nil {
		t.Fatal("expected unsupported resolution error")
	}
	if !strings.Contains(err.Error(), "supports only resolution=480p or 720p") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEstimateBillingUsesForced4KAliasBasePrice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Model: "Seedance-2.0-4K-海外版",
		Metadata: map[string]interface{}{
			"resolution": "4k",
			"content": []interface{}{
				map[string]interface{}{"type": "video_url", "video_url": map[string]interface{}{"url": "https://example.com/input.mp4"}},
			},
		},
	})

	ratios := (&TaskAdaptor{}).EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "Seedance-2.0-4K-海外版"})
	got, ok := ratios["seedance_unit_price"]
	if !ok {
		t.Fatal("expected seedance_unit_price ratio")
	}
	if got != 2.4/4.0 {
		t.Fatalf("seedance_unit_price = %v, want %v", got, 2.4/4.0)
	}
}

func TestEstimateBillingUsesSeedance15SilentUnitPrice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Model: "Seedance-1.5-pro-海外版",
		Metadata: map[string]interface{}{
			"resolution":     "720p",
			"generate_audio": false,
		},
	})

	ratios := (&TaskAdaptor{}).EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "Seedance-1.5-pro-海外版"})
	got, ok := ratios["seedance_unit_price"]
	if !ok {
		t.Fatal("expected seedance_unit_price ratio")
	}
	if got != 1.2/2.4 {
		t.Fatalf("seedance_unit_price = %v, want %v", got, 1.2/2.4)
	}
}

func TestConvertToRequestPayloadPreservesAssetReferences(t *testing.T) {
	request := relaycommon.TaskSubmitReq{
		Model:  "Seedance-2.0-海外版",
		Prompt: "保持已授权人物特征并参考视频动作",
		Metadata: map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]interface{}{
						"url": "asset://asset-image-1",
					},
					"role": "reference_image",
				},
				map[string]interface{}{
					"type": "video_url",
					"video_url": map[string]interface{}{
						"url": "asset://asset-video-1",
					},
					"role": "reference_video",
				},
			},
		},
	}

	payload, err := (&TaskAdaptor{}).convertToRequestPayload(&request)

	require.NoError(t, err)
	require.Len(t, payload.Content, 3)
	require.Equal(t, "image_url", payload.Content[0].Type)
	require.Equal(t, "asset://asset-image-1", payload.Content[0].ImageURL.URL)
	require.Equal(t, "reference_image", payload.Content[0].Role)
	require.Equal(t, "video_url", payload.Content[1].Type)
	require.Equal(t, "asset://asset-video-1", payload.Content[1].VideoURL.URL)
	require.Equal(t, "reference_video", payload.Content[1].Role)
	require.Equal(t, "text", payload.Content[2].Type)
	require.Equal(t, request.Prompt, payload.Content[2].Text)
}

func TestParseTaskResultFallsBackToCompletionTokens(t *testing.T) {
	adaptor := &TaskAdaptor{}
	result, err := adaptor.ParseTaskResult([]byte(`{
		"id": "task-1",
		"status": "succeeded",
		"content": {"video_url": "https://example.com/video.mp4"},
		"usage": {"completion_tokens": 1234}
	}`))
	if err != nil {
		t.Fatalf("ParseTaskResult returned error: %v", err)
	}
	if result.Status != model.TaskStatusSuccess {
		t.Fatalf("status = %v, want success", result.Status)
	}
	if result.CompletionTokens != 1234 {
		t.Fatalf("completion tokens = %d, want 1234", result.CompletionTokens)
	}
	if result.TotalTokens != 1234 {
		t.Fatalf("total tokens = %d, want completion token fallback 1234", result.TotalTokens)
	}
}

func TestParseTaskResultPrefersTotalTokens(t *testing.T) {
	adaptor := &TaskAdaptor{}
	result, err := adaptor.ParseTaskResult([]byte(`{
		"id": "task-1",
		"status": "succeeded",
		"usage": {"completion_tokens": 1234, "total_tokens": 5678}
	}`))
	if err != nil {
		t.Fatalf("ParseTaskResult returned error: %v", err)
	}
	if result.TotalTokens != 5678 {
		t.Fatalf("total tokens = %d, want 5678", result.TotalTokens)
	}
}
