package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCORSAllowsCrossOriginPatchRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(CORS())
	engine.PATCH("/api/seedance/actors/:actor_id", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodOptions, "/api/seedance/actors/1", nil)
	request.Header.Set("Origin", "https://www.token123.co")
	request.Header.Set("Access-Control-Request-Method", http.MethodPatch)
	recorder := httptest.NewRecorder()

	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.Contains(t, recorder.Header().Get("Access-Control-Allow-Methods"), http.MethodPatch)
}
