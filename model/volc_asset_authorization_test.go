package model

import (
	"encoding/hex"
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupVolcAssetAuthorizationTestDB(t *testing.T) {
	t.Helper()
	oldDB := DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&VolcAssetUserGroup{},
		&VolcAssetAuthorizationSession{},
	))
	DB = db
	t.Cleanup(func() { DB = oldDB })
}

func authorizationDigest(raw string) string {
	return hex.EncodeToString(common.Sha256Raw([]byte(raw)))
}

func TestStartVolcAssetAuthorizationExpiresOlderPendingSession(t *testing.T) {
	setupVolcAssetAuthorizationTestDB(t)
	firstDigest := authorizationDigest("raw-byted-token-first")
	secondDigest := authorizationDigest("raw-byted-token-second")

	first, err := StartVolcAssetAuthorization(42, firstDigest, 400, 100)
	require.NoError(t, err)
	second, err := StartVolcAssetAuthorization(42, secondDigest, 500, 200)
	require.NoError(t, err)

	var storedFirst VolcAssetAuthorizationSession
	require.NoError(t, DB.First(&storedFirst, first.Id).Error)
	require.Equal(t, VolcAssetAuthorizationExpired, storedFirst.Status)
	require.Equal(t, VolcAssetAuthorizationPending, second.Status)
	require.NotContains(t, second.TokenHash, "raw-byted-token")
	require.Len(t, second.TokenHash, 64)
}

func TestFindPendingVolcAssetAuthorizationRequiresSameUserAndDigest(t *testing.T) {
	setupVolcAssetAuthorizationTestDB(t)
	digest := authorizationDigest("raw-byted-token")
	require.NoError(t, mustStartVolcAssetAuthorization(42, digest, 400, 100))

	_, err := FindPendingVolcAssetAuthorization(43, digest, 200)
	require.ErrorIs(t, err, ErrVolcAssetAuthorizationNotFound)
	_, err = FindPendingVolcAssetAuthorization(42, authorizationDigest("other"), 200)
	require.ErrorIs(t, err, ErrVolcAssetAuthorizationNotFound)

	found, err := FindPendingVolcAssetAuthorization(42, digest, 200)
	require.NoError(t, err)
	require.Equal(t, 42, found.UserId)
}

func TestFindPendingVolcAssetAuthorizationRejectsExpiredSession(t *testing.T) {
	setupVolcAssetAuthorizationTestDB(t)
	digest := authorizationDigest("raw-byted-token")
	require.NoError(t, mustStartVolcAssetAuthorization(42, digest, 150, 100))

	_, err := FindPendingVolcAssetAuthorization(42, digest, 151)
	require.ErrorIs(t, err, ErrVolcAssetAuthorizationExpired)

	var stored VolcAssetAuthorizationSession
	require.NoError(t, DB.Where("user_id = ? AND token_hash = ?", 42, digest).First(&stored).Error)
	require.Equal(t, VolcAssetAuthorizationExpired, stored.Status)
}

func TestExpireVolcAssetAuthorizationConsumesOnlyMatchingPendingSession(t *testing.T) {
	setupVolcAssetAuthorizationTestDB(t)
	digest := authorizationDigest("raw-byted-token")
	require.NoError(t, mustStartVolcAssetAuthorization(42, digest, 400, 100))

	require.ErrorIs(t, ExpireVolcAssetAuthorization(43, digest, 200), ErrVolcAssetAuthorizationNotFound)
	require.NoError(t, ExpireVolcAssetAuthorization(42, digest, 200))
	_, err := FindPendingVolcAssetAuthorization(42, digest, 201)
	require.ErrorIs(t, err, ErrVolcAssetAuthorizationNotFound)
}

func TestCompleteVolcAssetAuthorizationBindsGroupAndConsumesSession(t *testing.T) {
	setupVolcAssetAuthorizationTestDB(t)
	digest := authorizationDigest("raw-byted-token")
	require.NoError(t, mustStartVolcAssetAuthorization(42, digest, 400, 100))

	require.NoError(t, CompleteVolcAssetAuthorization(42, digest, "group-real-42", 200))

	binding, err := GetVolcAssetUserGroup(42)
	require.NoError(t, err)
	require.Equal(t, "group-real-42", binding.GroupId)
	var stored VolcAssetAuthorizationSession
	require.NoError(t, DB.Where("user_id = ? AND token_hash = ?", 42, digest).First(&stored).Error)
	require.Equal(t, VolcAssetAuthorizationCompleted, stored.Status)
}

func TestCompleteVolcAssetAuthorizationIsIdempotentForSameBinding(t *testing.T) {
	setupVolcAssetAuthorizationTestDB(t)
	digest := authorizationDigest("raw-byted-token")
	require.NoError(t, mustStartVolcAssetAuthorization(42, digest, 400, 100))
	require.NoError(t, CompleteVolcAssetAuthorization(42, digest, "group-real-42", 200))

	require.NoError(t, CompleteVolcAssetAuthorization(42, digest, "group-other", 201))

	binding, err := GetVolcAssetUserGroup(42)
	require.NoError(t, err)
	require.Equal(t, "group-real-42", binding.GroupId)
}

func mustStartVolcAssetAuthorization(userID int, digest string, expiresAt, now int64) error {
	_, err := StartVolcAssetAuthorization(userID, digest, expiresAt, now)
	return err
}

func TestVolcAssetAuthorizationRejectsInvalidInput(t *testing.T) {
	setupVolcAssetAuthorizationTestDB(t)

	_, err := StartVolcAssetAuthorization(0, authorizationDigest("token"), 400, 100)
	require.Error(t, err)
	_, err = StartVolcAssetAuthorization(42, "raw-token", 400, 100)
	require.Error(t, err)
	_, err = FindPendingVolcAssetAuthorization(42, authorizationDigest("missing"), 100)
	require.True(t, errors.Is(err, ErrVolcAssetAuthorizationNotFound))
}
