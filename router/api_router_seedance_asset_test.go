package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSeedanceDashboardRoutesRequireUserAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	registerSeedanceAssetRoutes(engine.Group("/api"))

	want := map[string]bool{
		"GET /api/seedance/assets/authorization":          false,
		"POST /api/seedance/assets/authorization/session": false,
		"POST /api/seedance/assets/authorization/result":  false,
		"GET /api/seedance/assets":                        false,
		"POST /api/seedance/assets":                       false,
		"GET /api/seedance/assets/:id":                    false,
		"PATCH /api/seedance/assets/:id":                  false,
		"DELETE /api/seedance/assets/:id":                 false,
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

	for key := range want {
		parts := splitMethodAndPath(key)
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(parts[0], routeTestPath(parts[1]), nil)
		engine.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusUnauthorized, recorder.Code, key)
	}
}

func splitMethodAndPath(route string) [2]string {
	for i := 0; i < len(route); i++ {
		if route[i] == ' ' {
			return [2]string{route[:i], route[i+1:]}
		}
	}
	return [2]string{}
}

func routeTestPath(path string) string {
	if len(path) >= 3 && path[len(path)-3:] == ":id" {
		return path[:len(path)-3] + "asset-1"
	}
	return path
}
