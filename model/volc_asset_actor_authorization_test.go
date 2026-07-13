package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupVolcAssetActorAuthorizationTestDB(t *testing.T) {
	t.Helper()
	oldDB := DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&VolcAssetUserGroup{},
		&VolcAssetActor{},
		&VolcAssetActorAsset{},
		&VolcAssetAuthorizationSession{},
		&VolcAssetAuthorizationEvent{},
	))
	DB = db
	t.Cleanup(func() { DB = oldDB })
}

func TestActorAuthorizationSessionsRemainIndependent(t *testing.T) {
	setupVolcAssetActorAuthorizationTestDB(t)
	days := 30
	actorA, err := CreateVolcAssetActor(42, "演员甲", &days, 10)
	require.NoError(t, err)
	actorB, err := CreateVolcAssetActor(42, "演员乙", &days, 11)
	require.NoError(t, err)

	firstA, err := StartVolcAssetActorAuthorization(42, actorA.Id, authorizationDigest("byted-a1"), authorizationDigest("invite-a1"), authorizationDigest("revoke-a1"), &days, "v1", 500, 100)
	require.NoError(t, err)
	firstB, err := StartVolcAssetActorAuthorization(42, actorB.Id, authorizationDigest("byted-b1"), authorizationDigest("invite-b1"), authorizationDigest("revoke-b1"), &days, "v1", 500, 100)
	require.NoError(t, err)
	_, err = StartVolcAssetActorAuthorization(42, actorA.Id, authorizationDigest("byted-a2"), authorizationDigest("invite-a2"), authorizationDigest("revoke-a2"), &days, "v1", 600, 200)
	require.NoError(t, err)

	var storedA, storedB VolcAssetAuthorizationSession
	require.NoError(t, DB.First(&storedA, firstA.Id).Error)
	require.NoError(t, DB.First(&storedB, firstB.Id).Error)
	require.Equal(t, VolcAssetAuthorizationExpired, storedA.Status)
	require.Equal(t, VolcAssetAuthorizationPending, storedB.Status)

	_, err = FindPendingVolcAssetActorAuthorization(42, actorA.Id, firstB.TokenHash, 201)
	require.ErrorIs(t, err, ErrVolcAssetAuthorizationNotFound)
	_, err = FindPendingVolcAssetActorAuthorization(43, actorB.Id, firstB.TokenHash, 201)
	require.ErrorIs(t, err, ErrVolcAssetAuthorizationNotFound)
}

func TestActorAuthorizationRequiresConsentAndCalculatesExpiry(t *testing.T) {
	setupVolcAssetActorAuthorizationTestDB(t)
	days := 30
	actor, err := CreateVolcAssetActor(42, "演员甲", &days, 10)
	require.NoError(t, err)
	bytedHash := authorizationDigest("byted")
	inviteHash := authorizationDigest("invite")
	revokeHash := authorizationDigest("revoke")
	_, err = StartVolcAssetActorAuthorization(42, actor.Id, bytedHash, inviteHash, revokeHash, &days, "v1", 500, 100)
	require.NoError(t, err)

	err = CompleteVolcAssetActorAuthorization(42, actor.Id, bytedHash, "group-a", 200)
	require.ErrorIs(t, err, ErrVolcAssetConsentRequired)

	details, err := GetVolcAssetAuthorizationInvitation(inviteHash, 200)
	require.NoError(t, err)
	require.Equal(t, "演员甲", details.Actor.DisplayName)
	require.Equal(t, days, *details.Session.DurationDays)
	require.Empty(t, details.Actor.GroupId)

	accepted, err := AcceptVolcAssetAuthorizationConsent(inviteHash, 210)
	require.NoError(t, err)
	require.NotNil(t, accepted.ConsentAcceptedAt)
	acceptedAgain, err := AcceptVolcAssetAuthorizationConsent(inviteHash, 211)
	require.NoError(t, err)
	require.Equal(t, int64(210), *acceptedAgain.ConsentAcceptedAt)

	require.NoError(t, CompleteVolcAssetActorAuthorization(42, actor.Id, bytedHash, "group-a", 220))
	active, err := GetVolcAssetActorForUser(42, actor.Id)
	require.NoError(t, err)
	require.Equal(t, VolcAssetActorActive, active.Status)
	require.Equal(t, int64(220+30*86400), *active.AuthorizationExpiresAt)

	var events []VolcAssetAuthorizationEvent
	require.NoError(t, DB.Where("actor_id = ?", actor.Id).Order("id ASC").Find(&events).Error)
	require.Equal(t, []string{VolcAssetAuthorizationEventConsentAccepted, VolcAssetAuthorizationEventAuthorized}, []string{events[0].EventType, events[1].EventType})
}

func TestActorAuthorizationLongTermAndPublicRevocation(t *testing.T) {
	setupVolcAssetActorAuthorizationTestDB(t)
	actor, err := CreateVolcAssetActor(42, "长期演员", nil, 10)
	require.NoError(t, err)
	bytedHash := authorizationDigest("byted-long")
	inviteHash := authorizationDigest("invite-long")
	revokeHash := authorizationDigest("revoke-long")
	_, err = StartVolcAssetActorAuthorization(42, actor.Id, bytedHash, inviteHash, revokeHash, nil, "v1", 500, 100)
	require.NoError(t, err)
	_, err = AcceptVolcAssetAuthorizationConsent(inviteHash, 150)
	require.NoError(t, err)
	require.NoError(t, CompleteVolcAssetActorAuthorization(42, actor.Id, bytedHash, "group-long", 200))

	active, err := GetVolcAssetActorForUser(42, actor.Id)
	require.NoError(t, err)
	require.Nil(t, active.AuthorizationExpiresAt)
	require.NoError(t, RevokeVolcAssetActorByToken(revokeHash, 300))

	_, err = RequireVolcAssetActorActiveAt(42, actor.Id, 301)
	require.ErrorIs(t, err, ErrVolcAssetActorRevoked)
	var event VolcAssetAuthorizationEvent
	require.NoError(t, DB.Where("actor_id = ? AND event_type = ?", actor.Id, VolcAssetAuthorizationEventRevoked).First(&event).Error)
}
