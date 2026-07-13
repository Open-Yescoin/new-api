package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relay/channel/task/doubao"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeSeedanceAuthorizationService struct {
	status       *doubao.AuthorizationStatus
	session      *doubao.AuthorizationSession
	result       *doubao.AuthorizationResult
	err          error
	createUserId int
	pollUserId   int
	pollToken    string
}

func (f *fakeSeedanceAuthorizationService) Status(int) (*doubao.AuthorizationStatus, error) {
	return f.status, f.err
}

func (f *fakeSeedanceAuthorizationService) CreateSession(_ context.Context, userId int) (*doubao.AuthorizationSession, error) {
	f.createUserId = userId
	return f.session, f.err
}

func (f *fakeSeedanceAuthorizationService) Poll(_ context.Context, userId int, token string) (*doubao.AuthorizationResult, error) {
	f.pollUserId = userId
	f.pollToken = token
	return f.result, f.err
}

type fakeSeedanceAssetService struct {
	listResponse   *doubao.ListAssetsResponse
	assetResponse  *doubao.AssetItem
	createResponse *doubao.CreateAssetResponse
	err            error
	createRequest  doubao.CreateAssetRequest
}

func (f *fakeSeedanceAssetService) ListAssets(context.Context, int, doubao.ListAssetsRequest) (*doubao.ListAssetsResponse, error) {
	return f.listResponse, f.err
}

func (f *fakeSeedanceAssetService) CreateAsset(_ context.Context, _ int, request doubao.CreateAssetRequest) (*doubao.CreateAssetResponse, error) {
	f.createRequest = request
	return f.createResponse, f.err
}

func (f *fakeSeedanceAssetService) GetAsset(context.Context, int, doubao.GetAssetRequest) (*doubao.AssetItem, error) {
	return f.assetResponse, f.err
}

func (f *fakeSeedanceAssetService) UpdateAsset(context.Context, int, doubao.UpdateAssetRequest) error {
	return f.err
}

func (f *fakeSeedanceAssetService) DeleteAsset(context.Context, int, doubao.DeleteAssetRequest) error {
	return f.err
}

func performSeedanceControllerRequest(t *testing.T, method, path, body string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("id", 42)
		c.Next()
	})
	engine.Handle(method, path, handler)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)
	return recorder
}

func TestSeedanceAuthorizationStatusRedactsGroupID(t *testing.T) {
	authorization := &fakeSeedanceAuthorizationService{status: &doubao.AuthorizationStatus{
		Configured: true,
		Authorized: true,
	}}
	controller := newSeedanceAssetController(authorization, &fakeSeedanceAssetService{})

	recorder := performSeedanceControllerRequest(t, http.MethodGet, "/authorization", "", controller.GetAuthorizationStatus)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotContains(t, strings.ToLower(recorder.Body.String()), "group")
	var response struct {
		Success bool                       `json:"success"`
		Data    doubao.AuthorizationStatus `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.True(t, response.Data.Authorized)
}

func TestSeedanceCreateAuthorizationSessionUsesNoClientCallback(t *testing.T) {
	authorization := &fakeSeedanceAuthorizationService{session: &doubao.AuthorizationSession{
		BytedToken: "token-1",
		H5Link:     "https://verify.example/session",
		ExpiresAt:  400,
	}}
	controller := newSeedanceAssetController(authorization, &fakeSeedanceAssetService{})

	recorder := performSeedanceControllerRequest(
		t,
		http.MethodPost,
		"/authorization/session",
		`{"callback_url":"https://attacker.example/callback"}`,
		controller.CreateAuthorizationSession,
	)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, 42, authorization.createUserId)
}

func TestSeedancePollAuthorizationRequiresBytedToken(t *testing.T) {
	authorization := &fakeSeedanceAuthorizationService{}
	controller := newSeedanceAssetController(authorization, &fakeSeedanceAssetService{})

	recorder := performSeedanceControllerRequest(t, http.MethodPost, "/authorization/result", `{}`, controller.GetAuthorizationResult)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Zero(t, authorization.pollUserId)
	require.Contains(t, recorder.Body.String(), "invalid_request")
}

func TestSeedanceAssetCreateIgnoresClientGroupID(t *testing.T) {
	assets := &fakeSeedanceAssetService{createResponse: &doubao.CreateAssetResponse{Id: "asset-1"}}
	controller := newSeedanceAssetController(&fakeSeedanceAuthorizationService{}, assets)

	recorder := performSeedanceControllerRequest(
		t,
		http.MethodPost,
		"/",
		`{"url":"https://cdn.example/person.jpg","asset_type":"Image","group_id":"group-attacker"}`,
		controller.CreateAsset,
	)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Empty(t, assets.createRequest.GroupId)
	require.Equal(t, "https://cdn.example/person.jpg", assets.createRequest.URL)
	require.Equal(t, "Image", assets.createRequest.AssetType)
}

func TestSeedanceAssetNotOwnedReturns404(t *testing.T) {
	assets := &fakeSeedanceAssetService{err: doubao.ErrAssetNotFound}
	controller := newSeedanceAssetController(&fakeSeedanceAuthorizationService{}, assets)

	recorder := performSeedanceControllerRequest(t, http.MethodGet, "/asset-other", "", controller.GetAsset)

	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Contains(t, recorder.Body.String(), "asset_not_found")
}
