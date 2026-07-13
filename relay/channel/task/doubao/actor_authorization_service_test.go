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

func setupActorAuthorizationServiceTest(t *testing.T) (*ActorAuthorizationService, *fakeAssetAPI) {
	t.Helper()
	oldDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.VolcAssetActor{},
		&model.VolcAssetActorAsset{},
		&model.VolcAssetAuthorizationSession{},
		&model.VolcAssetAuthorizationEvent{},
	))
	model.DB = db
	t.Cleanup(func() { model.DB = oldDB })
	api := &fakeAssetAPI{responses: map[string]any{}, errors: map[string]error{}}
	config := system_setting.VolcAssetSettings{
		AccessKey:                    "ak",
		SecretKey:                    "sk",
		ProjectName:                  "project-a",
		AuthorizationCallbackBaseURL: "https://www.token123.co",
	}
	service := NewActorAuthorizationService(api, config, func() time.Time { return time.Unix(100, 0) })
	return service, api
}

func TestActorAuthorizationServiceCreatesSafeHandoffAndCompletesConsent(t *testing.T) {
	service, api := setupActorAuthorizationServiceTest(t)
	days := 30
	actor, err := model.CreateVolcAssetActor(42, "演员甲", &days, 10)
	require.NoError(t, err)
	api.responses["CreateVisualValidateSession"] = CreateVisualValidateSessionResponse{
		BytedToken: "byted-secret",
		H5Link:     "https://verify.example/h5",
	}

	session, err := service.CreateSession(context.Background(), 42, actor.Id)
	require.NoError(t, err)
	require.NotEmpty(t, session.InvitationToken)
	require.NotEmpty(t, session.RevocationToken)
	require.NotEqual(t, session.InvitationToken, session.RevocationToken)

	details, err := service.InvitationDetails(session.InvitationToken)
	require.NoError(t, err)
	require.Equal(t, "演员甲", details.ActorName)
	require.Equal(t, &days, details.DurationDays)
	require.Equal(t, actor.Id, details.ActorId)

	require.NoError(t, service.AcceptConsent(session.InvitationToken))
	api.responses["GetVisualValidateResult"] = GetVisualValidateResultResponse{GroupId: "group-a"}
	result, err := service.Poll(context.Background(), 42, actor.Id, session.BytedToken)
	require.NoError(t, err)
	require.True(t, result.Authorized)

	active, err := model.GetVolcAssetActorForUser(42, actor.Id)
	require.NoError(t, err)
	require.Equal(t, int64(100+30*86400), *active.AuthorizationExpiresAt)
	require.NotContains(t, session.InvitationToken, "byted-secret")

	request := api.calls[0].request.(CreateVisualValidateSessionRequest)
	require.Equal(t, "https://www.token123.co/seedance/authorization/callback", request.CallbackURL)
}

func TestActorAuthorizationServiceKeepsActorsIndependentAndRevokesByToken(t *testing.T) {
	service, api := setupActorAuthorizationServiceTest(t)
	actorA, err := model.CreateVolcAssetActor(42, "演员甲", nil, 10)
	require.NoError(t, err)
	actorB, err := model.CreateVolcAssetActor(42, "演员乙", nil, 11)
	require.NoError(t, err)
	api.responses["CreateVisualValidateSession"] = CreateVisualValidateSessionResponse{BytedToken: "byted-a", H5Link: "https://verify.example/a"}
	sessionA, err := service.CreateSession(context.Background(), 42, actorA.Id)
	require.NoError(t, err)
	require.NoError(t, service.AcceptConsent(sessionA.InvitationToken))
	api.responses["GetVisualValidateResult"] = GetVisualValidateResultResponse{GroupId: "group-a"}
	_, err = service.Poll(context.Background(), 42, actorA.Id, sessionA.BytedToken)
	require.NoError(t, err)

	api.responses["CreateVisualValidateSession"] = CreateVisualValidateSessionResponse{BytedToken: "byted-b", H5Link: "https://verify.example/b"}
	_, err = service.CreateSession(context.Background(), 42, actorB.Id)
	require.NoError(t, err)
	require.NoError(t, service.RevokeByToken(sessionA.RevocationToken))

	_, err = model.RequireVolcAssetActorActiveAt(42, actorA.Id, 101)
	require.ErrorIs(t, err, model.ErrVolcAssetActorRevoked)
	pendingB, err := model.ListVolcAssetActors(42)
	require.NoError(t, err)
	require.Equal(t, model.VolcAssetActorPending, pendingB[1].Status)
}
