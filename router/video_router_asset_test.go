package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestVideoRouterRegistersSeedanceAssetLibraryRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetVideoRouter(engine)

	want := map[string]bool{
		"POST /doubao/open/CreateVisualValidateSession": false,
		"POST /doubao/open/GetVisualValidateResult":     false,
		"POST /doubao/open/ListAssets":                  false,
		"POST /doubao/open/GetAsset":                    false,
		"POST /doubao/open/CreateAsset":                 false,
		"POST /doubao/open/UpdateAsset":                 false,
		"POST /doubao/open/DeleteAsset":                 false,
	}
	for _, route := range engine.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, registered := range want {
		require.True(t, registered, "missing route %s", route)
	}
}

func TestSeedanceAssetLibraryRoutesRequireToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetVideoRouter(engine)

	paths := []string{
		"/doubao/open/CreateVisualValidateSession",
		"/doubao/open/GetVisualValidateResult",
		"/doubao/open/ListAssets",
		"/doubao/open/GetAsset",
		"/doubao/open/CreateAsset",
		"/doubao/open/UpdateAsset",
		"/doubao/open/DeleteAsset",
	}
	for _, path := range paths {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, path, nil)

		engine.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusUnauthorized, recorder.Code, path)
	}
}
