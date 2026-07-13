package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupVolcAssetActorTestDB(t *testing.T) {
	t.Helper()
	oldDB := DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&VolcAssetUserGroup{},
		&VolcAssetActor{},
		&VolcAssetActorAsset{},
	))
	DB = db
	t.Cleanup(func() { DB = oldDB })
}

func TestVolcAssetActorSupportsApprovedDurationsAndMultipleActors(t *testing.T) {
	setupVolcAssetActorTestDB(t)

	for _, days := range []int{30, 60, 180, 365} {
		actor, err := CreateVolcAssetActor(42, "演员", &days, int64(100+days))
		require.NoError(t, err)
		require.Equal(t, days, *actor.AuthorizationDurationDays)
	}
	longTerm, err := CreateVolcAssetActor(42, "长期演员", nil, 999)
	require.NoError(t, err)
	require.Nil(t, longTerm.AuthorizationDurationDays)

	actors, err := ListVolcAssetActors(42)
	require.NoError(t, err)
	require.Len(t, actors, 5)

	invalidDays := 31
	_, err = CreateVolcAssetActor(42, "无效演员", &invalidDays, 1000)
	require.ErrorIs(t, err, ErrVolcAssetInvalidDuration)
}

func TestVolcAssetActorOwnershipAndActiveBoundary(t *testing.T) {
	setupVolcAssetActorTestDB(t)
	days := 30
	actor, err := CreateVolcAssetActor(42, "演员甲", &days, 100)
	require.NoError(t, err)

	_, err = GetVolcAssetActorForUser(43, actor.Id)
	require.ErrorIs(t, err, ErrVolcAssetActorNotFound)

	require.NoError(t, ActivateVolcAssetActor(42, actor.Id, "group-a", "v1", 200))
	active, err := RequireVolcAssetActorActiveAt(42, actor.Id, 200+30*86400-1)
	require.NoError(t, err)
	require.Equal(t, "group-a", active.GroupId)

	_, err = RequireVolcAssetActorActiveAt(42, actor.Id, 200+30*86400)
	require.ErrorIs(t, err, ErrVolcAssetActorExpired)

	stored, err := GetVolcAssetActorForUser(42, actor.Id)
	require.NoError(t, err)
	require.Equal(t, VolcAssetActorExpired, stored.Status)
}

func TestVolcAssetActorRevocationAndAssetMappingAreIsolated(t *testing.T) {
	setupVolcAssetActorTestDB(t)
	actorA, err := CreateVolcAssetActor(42, "演员甲", nil, 100)
	require.NoError(t, err)
	actorB, err := CreateVolcAssetActor(42, "演员乙", nil, 101)
	require.NoError(t, err)
	require.NoError(t, ActivateVolcAssetActor(42, actorA.Id, "group-a", "v1", 200))
	require.NoError(t, ActivateVolcAssetActor(42, actorB.Id, "group-b", "v1", 200))
	require.NoError(t, SaveVolcAssetActorAsset(42, actorA.Id, "asset-a", 201))
	require.NoError(t, SaveVolcAssetActorAsset(42, actorB.Id, "asset-b", 201))

	mapped, err := FindVolcAssetActorByAsset(42, "asset-b")
	require.NoError(t, err)
	require.Equal(t, actorB.Id, mapped.Id)
	_, err = FindVolcAssetActorByAsset(43, "asset-b")
	require.ErrorIs(t, err, ErrVolcAssetActorNotFound)

	require.NoError(t, RevokeVolcAssetActor(42, actorA.Id, 300))
	_, err = RequireVolcAssetActorActiveAt(42, actorA.Id, 301)
	require.ErrorIs(t, err, ErrVolcAssetActorRevoked)
	_, err = RequireVolcAssetActorActiveAt(42, actorB.Id, 301)
	require.NoError(t, err)
}

func TestMigrateVolcAssetUserGroupsToActorsIsIdempotent(t *testing.T) {
	setupVolcAssetActorTestDB(t)
	require.NoError(t, SaveVolcAssetUserGroup(42, "legacy-group"))

	require.NoError(t, MigrateVolcAssetUserGroupsToActors())
	require.NoError(t, MigrateVolcAssetUserGroupsToActors())

	actors, err := ListVolcAssetActors(42)
	require.NoError(t, err)
	require.Len(t, actors, 1)
	require.True(t, actors[0].IsDefault)
	require.Equal(t, "legacy-group", actors[0].GroupId)
	require.Equal(t, VolcAssetActorLegacyReview, actors[0].Status)
}
