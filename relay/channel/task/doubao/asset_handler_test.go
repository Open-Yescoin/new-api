package doubao

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newAssetHandlerContext(body string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/doubao/open/action", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", 42)
	return c, recorder
}

func TestAssetHandlerRejectsInvalidJSON(t *testing.T) {
	service, _, _ := newAssetServiceTestFixture()
	handler := NewAssetHandler(service)
	c, recorder := newAssetHandlerContext("{")

	handler.CreateAsset(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestAssetHandlerCreatesValidationSession(t *testing.T) {
	service, api, _ := newAssetServiceTestFixture()
	api.responses["CreateVisualValidateSession"] = CreateVisualValidateSessionResponse{
		BytedToken: "token-1",
		H5Link:     "https://verify.example/session",
	}
	handler := NewAssetHandler(service)
	c, recorder := newAssetHandlerContext(`{"CallbackURL":"https://app.example/callback"}`)

	handler.CreateVisualValidateSession(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "token-1")
	require.NotContains(t, recorder.Body.String(), "project-a")
}

func TestAssetHandlerCreatesAsset(t *testing.T) {
	service, api, groups := newAssetServiceTestFixture()
	groups.groups[42] = "group-real-42"
	api.responses["CreateAsset"] = CreateAssetResponse{Id: "asset-1", Status: "Processing"}
	handler := NewAssetHandler(service)
	c, recorder := newAssetHandlerContext(`{"URL":"https://cdn.example/person.jpg","AssetType":"Image"}`)

	handler.CreateAsset(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response CreateAssetResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "asset-1", response.Id)
}

func TestAssetHandlerRequiresAuthorizationGroup(t *testing.T) {
	service, _, _ := newAssetServiceTestFixture()
	handler := NewAssetHandler(service)
	c, recorder := newAssetHandlerContext(`{"PageNumber":1,"PageSize":20}`)

	handler.ListAssets(c)

	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Contains(t, recorder.Body.String(), "real-person asset authorization is required")
}

func TestAssetHandlerHidesCrossGroupAsset(t *testing.T) {
	service, api, groups := newAssetServiceTestFixture()
	groups.groups[42] = "group-real-42"
	api.responses["GetAsset"] = AssetItem{Id: "asset-other", GroupId: "group-other"}
	handler := NewAssetHandler(service)
	c, recorder := newAssetHandlerContext(`{"Id":"asset-other"}`)

	handler.GetAsset(c)

	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestAssetHandlerMapsConfigurationAndTransportErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
	}{
		{name: "configuration", err: ErrVolcAssetNotConfigured, statusCode: http.StatusServiceUnavailable},
		{name: "transport", err: errors.New("connection failed"), statusCode: http.StatusBadGateway},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, api, _ := newAssetServiceTestFixture()
			api.errors["CreateVisualValidateSession"] = test.err
			handler := NewAssetHandler(service)
			c, recorder := newAssetHandlerContext(`{"CallbackURL":"https://app.example/callback"}`)

			handler.CreateVisualValidateSession(c)

			require.Equal(t, test.statusCode, recorder.Code)
		})
	}
}
