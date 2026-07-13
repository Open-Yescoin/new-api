package doubao

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupActorAssetServiceTest(t *testing.T) (*AssetService, *fakeAssetAPI) {
	t.Helper()
	oldDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.VolcAssetActor{},
		&model.VolcAssetActorAsset{},
	))
	model.DB = db
	t.Cleanup(func() { model.DB = oldDB })
	api := &fakeAssetAPI{responses: map[string]any{}, errors: map[string]error{}}
	groups := &fakeAssetGroupRepository{groups: map[int]string{}, updatedAt: map[int]int64{}}
	service := NewAssetService(api, groups, system_setting.VolcAssetSettings{ProjectName: "project-a", GroupType: "LivenessFace"})
	service.now = func() time.Time { return time.Unix(100, 0) }
	return service, api
}

func TestActorAssetServiceScopesCreateAndListAndSavesMappings(t *testing.T) {
	service, api := setupActorAssetServiceTest(t)
	actorA, err := model.CreateVolcAssetActor(42, "演员甲", nil, 10)
	require.NoError(t, err)
	actorB, err := model.CreateVolcAssetActor(42, "演员乙", nil, 11)
	require.NoError(t, err)
	require.NoError(t, model.ActivateVolcAssetActor(42, actorA.Id, "group-a", "v1", 20))
	require.NoError(t, model.ActivateVolcAssetActor(42, actorB.Id, "group-b", "v1", 20))
	api.responses["CreateAsset"] = CreateAssetResponse{Id: "asset-a", Status: "Processing"}

	_, err = service.CreateActorAsset(context.Background(), 42, actorA.Id, CreateAssetRequest{URL: "https://cdn.example/a.jpg", AssetType: "Image"})
	require.NoError(t, err)
	request := api.calls[0].request.(CreateAssetRequest)
	require.Equal(t, "group-a", request.GroupId)
	mapped, err := model.FindVolcAssetActorByAsset(42, "asset-a")
	require.NoError(t, err)
	require.Equal(t, actorA.Id, mapped.Id)

	api.responses["ListAssets"] = ListAssetsResponse{Items: []AssetItem{{Id: "asset-b", GroupId: "group-b"}}, TotalCount: 7}
	listed, err := service.ListActorAssets(context.Background(), 42, actorB.Id, ListAssetsRequest{})
	require.NoError(t, err)
	require.EqualValues(t, 7, listed.TotalCount)
	mapped, err = model.FindVolcAssetActorByAsset(42, "asset-b")
	require.NoError(t, err)
	require.Equal(t, actorB.Id, mapped.Id)
}

func TestActorAssetServiceBlocksExpiredCreateButAllowsCleanup(t *testing.T) {
	service, api := setupActorAssetServiceTest(t)
	days := 30
	actor, err := model.CreateVolcAssetActor(42, "演员甲", &days, 10)
	require.NoError(t, err)
	require.NoError(t, model.ActivateVolcAssetActor(42, actor.Id, "group-a", "v1", 100-30*86400))

	_, err = service.CreateActorAsset(context.Background(), 42, actor.Id, CreateAssetRequest{URL: "https://cdn.example/a.jpg", AssetType: "Image"})
	require.ErrorIs(t, err, model.ErrVolcAssetActorExpired)
	require.Empty(t, api.calls)

	api.responses["GetAsset"] = AssetItem{Id: "asset-a", GroupId: "group-a"}
	_, err = service.GetActorAsset(context.Background(), 42, actor.Id, GetAssetRequest{Id: "asset-a"})
	require.NoError(t, err)
	require.NoError(t, service.UpdateActorAsset(context.Background(), 42, actor.Id, UpdateAssetRequest{Id: "asset-a", Name: "新名称"}))
	require.NoError(t, service.DeleteActorAsset(context.Background(), 42, actor.Id, DeleteAssetRequest{Id: "asset-a"}))
}

func TestActorAssetServiceRejectsCrossActorAsset(t *testing.T) {
	service, api := setupActorAssetServiceTest(t)
	actorA, err := model.CreateVolcAssetActor(42, "演员甲", nil, 10)
	require.NoError(t, err)
	require.NoError(t, model.ActivateVolcAssetActor(42, actorA.Id, "group-a", "v1", 20))
	api.responses["GetAsset"] = AssetItem{Id: "asset-b", GroupId: "group-b"}

	_, err = service.GetActorAsset(context.Background(), 42, actorA.Id, GetAssetRequest{Id: "asset-b"})
	require.ErrorIs(t, err, ErrAssetNotFound)
}
