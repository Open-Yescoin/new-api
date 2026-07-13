package doubao

import (
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupActorServiceTest(t *testing.T) *ActorService {
	t.Helper()
	oldDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.VolcAssetActor{},
		&model.VolcAssetActorAsset{},
		&model.VolcAssetAuthorizationSession{},
	))
	model.DB = db
	t.Cleanup(func() { model.DB = oldDB })
	return NewActorService(func() time.Time { return time.Unix(100, 0) })
}

func TestActorServiceCreatesListsAndMasksGroupID(t *testing.T) {
	service := setupActorServiceTest(t)
	days := 30
	first, err := service.Create(42, "演员", &days)
	require.NoError(t, err)
	second, err := service.Create(42, "演员", nil)
	require.NoError(t, err)
	require.NotEqual(t, first.Id, second.Id)
	require.NoError(t, model.ActivateVolcAssetActor(42, first.Id, "secret-group", "v1", 100))

	actors, err := service.List(42)
	require.NoError(t, err)
	require.Len(t, actors, 2)
	data, err := common.Marshal(actors)
	require.NoError(t, err)
	require.NotContains(t, strings.ToLower(string(data)), "group")
}

func TestActorServiceMaterializesExpiryAndIsolatesOwnership(t *testing.T) {
	service := setupActorServiceTest(t)
	days := 30
	actor, err := service.Create(42, "演员甲", &days)
	require.NoError(t, err)
	require.NoError(t, model.ActivateVolcAssetActor(42, actor.Id, "group-a", "v1", 100-30*86400))

	actors, err := service.List(42)
	require.NoError(t, err)
	require.Equal(t, model.VolcAssetActorExpired, actors[0].Status)
	require.ErrorIs(t, service.Rename(43, actor.Id, "越权"), model.ErrVolcAssetActorNotFound)
	require.ErrorIs(t, service.Revoke(43, actor.Id), model.ErrVolcAssetActorNotFound)
}

func TestActorServiceDeletesOnlyUnusedPendingActor(t *testing.T) {
	service := setupActorServiceTest(t)
	pending, err := service.Create(42, "待授权", nil)
	require.NoError(t, err)
	require.NoError(t, service.Delete(42, pending.Id))

	active, err := service.Create(42, "已授权", nil)
	require.NoError(t, err)
	require.NoError(t, model.ActivateVolcAssetActor(42, active.Id, "group-a", "v1", 100))
	require.ErrorIs(t, service.Delete(42, active.Id), model.ErrVolcAssetActorInUse)
}
