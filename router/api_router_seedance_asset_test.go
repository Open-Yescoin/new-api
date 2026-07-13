package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
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
		"PUT /api/seedance/assets/:id":                    false,
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

func TestSeedanceActorAndPublicAuthorizationRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(sessions.Sessions("session", cookie.NewStore([]byte("test-secret"))))
	registerSeedanceAssetRoutes(engine.Group("/api"))

	authenticated := []string{
		"GET /api/seedance/actors",
		"POST /api/seedance/actors",
		"PUT /api/seedance/actors/:actor_id",
		"PATCH /api/seedance/actors/:actor_id",
		"DELETE /api/seedance/actors/:actor_id",
		"GET /api/seedance/actors/:actor_id/authorization",
		"POST /api/seedance/actors/:actor_id/authorization/session",
		"POST /api/seedance/actors/:actor_id/authorization/result",
		"POST /api/seedance/actors/:actor_id/authorization/revoke",
		"GET /api/seedance/actors/:actor_id/assets",
		"POST /api/seedance/actors/:actor_id/assets",
		"GET /api/seedance/actors/:actor_id/assets/:id",
		"PUT /api/seedance/actors/:actor_id/assets/:id",
		"PATCH /api/seedance/actors/:actor_id/assets/:id",
		"DELETE /api/seedance/actors/:actor_id/assets/:id",
	}
	public := []string{
		"POST /api/seedance/public/authorization/details",
		"POST /api/seedance/public/authorization/consent",
		"POST /api/seedance/public/authorization/revoke",
	}
	routes := map[string]bool{}
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = true
	}
	for _, route := range append(authenticated, public...) {
		require.True(t, routes[route], "missing route %s", route)
	}
	for _, route := range authenticated {
		parts := splitMethodAndPath(route)
		path := strings.ReplaceAll(parts[1], ":actor_id", "1")
		path = strings.ReplaceAll(path, ":id", "asset-1")
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(parts[0], path, nil)
		engine.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusUnauthorized, recorder.Code, route)
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
