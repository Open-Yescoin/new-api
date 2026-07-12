package model

import (
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupVolcAssetGroupTestDB(t *testing.T) {
	t.Helper()
	oldDB := DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&VolcAssetUserGroup{}))
	DB = db
	t.Cleanup(func() { DB = oldDB })
}

func TestGetVolcAssetUserGroupReturnsNotFound(t *testing.T) {
	setupVolcAssetGroupTestDB(t)

	_, err := GetVolcAssetUserGroup(42)

	require.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestSaveAndGetVolcAssetUserGroup(t *testing.T) {
	setupVolcAssetGroupTestDB(t)

	require.NoError(t, SaveVolcAssetUserGroup(42, "group-42"))
	got, err := GetVolcAssetUserGroup(42)

	require.NoError(t, err)
	require.Equal(t, "group-42", got.GroupId)
}

func TestSaveVolcAssetUserGroupKeepsFirstBinding(t *testing.T) {
	setupVolcAssetGroupTestDB(t)

	require.NoError(t, SaveVolcAssetUserGroup(42, "group-first"))
	require.NoError(t, SaveVolcAssetUserGroup(42, "group-second"))
	got, err := GetVolcAssetUserGroup(42)

	require.NoError(t, err)
	require.Equal(t, "group-first", got.GroupId)
}

func TestSaveVolcAssetUserGroupRejectsInvalidInput(t *testing.T) {
	setupVolcAssetGroupTestDB(t)

	require.Error(t, SaveVolcAssetUserGroup(0, "group"))
	require.Error(t, SaveVolcAssetUserGroup(42, ""))
}
