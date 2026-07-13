package doubao

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/require"
)

type fakeAssetAuthorizationRepository struct {
	userId         int
	tokenHash      string
	expiresAt      int64
	startNow       int64
	pending        bool
	expired        bool
	completedGroup string
}

func (f *fakeAssetAuthorizationRepository) Start(userId int, tokenHash string, expiresAt, now int64) error {
	f.userId = userId
	f.tokenHash = tokenHash
	f.expiresAt = expiresAt
	f.startNow = now
	f.pending = true
	return nil
}

func (f *fakeAssetAuthorizationRepository) FindPending(userId int, tokenHash string, now int64) error {
	if !f.pending || f.userId != userId || f.tokenHash != tokenHash {
		return model.ErrVolcAssetAuthorizationNotFound
	}
	if f.expiresAt <= now {
		return model.ErrVolcAssetAuthorizationExpired
	}
	return nil
}

func (f *fakeAssetAuthorizationRepository) HasPending(userId int, now int64) (bool, error) {
	return f.pending && f.userId == userId && f.expiresAt > now, nil
}

func (f *fakeAssetAuthorizationRepository) Expire(userId int, tokenHash string, _ int64) error {
	if f.userId != userId || f.tokenHash != tokenHash {
		return model.ErrVolcAssetAuthorizationNotFound
	}
	f.pending = false
	f.expired = true
	return nil
}

func (f *fakeAssetAuthorizationRepository) Complete(userId int, tokenHash, groupId string, _ int64) error {
	if !f.pending || f.userId != userId || f.tokenHash != tokenHash {
		return model.ErrVolcAssetAuthorizationNotFound
	}
	f.pending = false
	f.completedGroup = groupId
	return nil
}

func fixedAuthorizationTime() time.Time {
	return time.Unix(100, 0)
}

func authorizationServiceFixture() (*AuthorizationService, *fakeAssetAPI, *fakeAssetGroupRepository, *fakeAssetAuthorizationRepository) {
	api := &fakeAssetAPI{responses: map[string]any{}, errors: map[string]error{}}
	groups := &fakeAssetGroupRepository{groups: map[int]string{}, updatedAt: map[int]int64{}}
	sessions := &fakeAssetAuthorizationRepository{}
	config := system_setting.VolcAssetSettings{
		AccessKey:                    "ak",
		SecretKey:                    "sk",
		ProjectName:                  "project-a",
		AuthorizationCallbackBaseURL: "https://www.token123.co",
	}
	return NewAuthorizationService(api, groups, sessions, config, fixedAuthorizationTime), api, groups, sessions
}

func TestAuthorizationStatusDoesNotReturnGroupID(t *testing.T) {
	service, api, groups, _ := authorizationServiceFixture()
	groups.groups[42] = "group-secret-42"
	groups.updatedAt[42] = 123

	status, err := service.Status(42)

	require.NoError(t, err)
	require.True(t, status.Configured)
	require.True(t, status.Authorized)
	require.Equal(t, int64(123), status.UpdatedAt)
	data, err := common.Marshal(status)
	require.NoError(t, err)
	require.NotContains(t, strings.ToLower(string(data)), "group")
	require.Empty(t, api.calls)
}

func TestCreateAuthorizationSessionUsesConfiguredCallbackAndStoresDigest(t *testing.T) {
	service, api, _, sessions := authorizationServiceFixture()
	api.responses["CreateVisualValidateSession"] = CreateVisualValidateSessionResponse{
		BytedToken: " raw-byted-token ",
		H5Link:     "https://verify.example/session",
	}

	result, err := service.CreateSession(context.Background(), 42)

	require.NoError(t, err)
	require.Equal(t, "raw-byted-token", result.BytedToken)
	require.Equal(t, int64(1000), result.ExpiresAt)
	require.Equal(t, 42, sessions.userId)
	require.Equal(t, int64(1000), sessions.expiresAt)
	require.NotEqual(t, result.BytedToken, sessions.tokenHash)
	require.Equal(t, hex.EncodeToString(common.Sha256Raw([]byte(result.BytedToken))), sessions.tokenHash)
	request := api.calls[0].request.(CreateVisualValidateSessionRequest)
	require.Equal(t, "https://www.token123.co/seedance/authorization/callback", request.CallbackURL)
	require.Equal(t, "project-a", request.ProjectName)
}

func TestCreateAuthorizationSessionRejectsMissingH5Data(t *testing.T) {
	service, api, _, sessions := authorizationServiceFixture()
	api.responses["CreateVisualValidateSession"] = CreateVisualValidateSessionResponse{BytedToken: "token-only"}

	_, err := service.CreateSession(context.Background(), 42)

	require.ErrorIs(t, err, ErrInvalidAssetAuthorizationResponse)
	require.Empty(t, sessions.tokenHash)
}

func TestCreateAuthorizationSessionRejectsNonHTTPSH5Link(t *testing.T) {
	service, api, _, sessions := authorizationServiceFixture()
	api.responses["CreateVisualValidateSession"] = CreateVisualValidateSessionResponse{
		BytedToken: "token-1",
		H5Link:     "http://verify.example/session",
	}

	_, err := service.CreateSession(context.Background(), 42)

	require.ErrorIs(t, err, ErrInvalidAssetAuthorizationResponse)
	require.Empty(t, sessions.tokenHash)
}

func TestPollAuthorizationRejectsAnotherUsersDigest(t *testing.T) {
	service, api, _, sessions := authorizationServiceFixture()
	rawToken := "raw-byted-token"
	sessions.Start(42, hex.EncodeToString(common.Sha256Raw([]byte(rawToken))), 400, 100)

	_, err := service.Poll(context.Background(), 43, rawToken)

	require.ErrorIs(t, err, model.ErrVolcAssetAuthorizationNotFound)
	require.Empty(t, api.calls)
}

func TestPollAuthorizationReturnsPendingForEmptyGroup(t *testing.T) {
	service, api, _, sessions := authorizationServiceFixture()
	rawToken := "raw-byted-token"
	sessions.Start(42, hex.EncodeToString(common.Sha256Raw([]byte(rawToken))), 400, 100)
	api.responses["GetVisualValidateResult"] = GetVisualValidateResultResponse{}

	result, err := service.Poll(context.Background(), 42, rawToken)

	require.NoError(t, err)
	require.Equal(t, AuthorizationResult{Status: "pending", Authorized: false}, *result)
	require.True(t, sessions.pending)
}

func TestPollAuthorizationExpiresObserved40004(t *testing.T) {
	service, api, _, sessions := authorizationServiceFixture()
	rawToken := "raw-byted-token"
	sessions.Start(42, hex.EncodeToString(common.Sha256Raw([]byte(rawToken))), 400, 100)
	api.errors["GetVisualValidateResult"] = &AssetAPIError{Code: "40004", Message: "The current link has expired"}

	_, err := service.Poll(context.Background(), 42, rawToken)

	require.ErrorIs(t, err, ErrAssetAuthorizationExpired)
	require.True(t, sessions.expired)
}

func TestPollAuthorizationCompletesBinding(t *testing.T) {
	service, api, _, sessions := authorizationServiceFixture()
	rawToken := "raw-byted-token"
	sessions.Start(42, hex.EncodeToString(common.Sha256Raw([]byte(rawToken))), 400, 100)
	api.responses["GetVisualValidateResult"] = GetVisualValidateResultResponse{GroupId: "group-real-42"}

	result, err := service.Poll(context.Background(), 42, rawToken)

	require.NoError(t, err)
	require.Equal(t, AuthorizationResult{Status: "completed", Authorized: true}, *result)
	require.Equal(t, "group-real-42", sessions.completedGroup)
}

func TestAuthorizationServiceMapsExpiredLocalSession(t *testing.T) {
	service, _, _, _ := authorizationServiceFixture()

	_, err := service.Poll(context.Background(), 42, "missing")

	require.True(t, errors.Is(err, model.ErrVolcAssetAuthorizationNotFound))
}
