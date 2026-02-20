package clockwork

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestClockworkRoute_HeaderIDStrictOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := Config{Enabled: true, HeaderName: "X-Clockwork", IDHeader: "X-Clockwork-Id"}
	cfg.Normalize()
	store := NewInMemoryStorage(10, 1024*1024)
	cw := NewClockwork(cfg, store)

	headerMeta := &Metadata{ID: "header-id", Method: "GET", URI: "/header"}
	routeMeta := &Metadata{ID: "route-id", Method: "GET", URI: "/route"}
	require.NoError(t, cw.SaveMetadata(context.Background(), headerMeta))
	require.NoError(t, cw.SaveMetadata(context.Background(), routeMeta))

	router := gin.New()
	RegisterRoutes(router, cw, nil)

	req := httptest.NewRequest(http.MethodGet, "/__clockwork/route-id", nil)
	req.Header.Set("X-Clockwork-Id", "header-id")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code)

	var got Metadata
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &got))
	require.Equal(t, "header-id", got.ID)
}

func TestClockworkRoute_HeaderIDStrictOverrideNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := Config{Enabled: true, HeaderName: "X-Clockwork", IDHeader: "X-Clockwork-Id"}
	cfg.Normalize()
	store := NewInMemoryStorage(10, 1024*1024)
	cw := NewClockwork(cfg, store)

	require.NoError(t, cw.SaveMetadata(context.Background(), &Metadata{ID: "route-id", Method: "GET", URI: "/route"}))

	router := gin.New()
	RegisterRoutes(router, cw, nil)

	req := httptest.NewRequest(http.MethodGet, "/__clockwork/route-id", nil)
	req.Header.Set("X-Clockwork-Id", "missing-id")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusNotFound, res.Code)
}
