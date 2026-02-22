package gin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RezaKargar/go-clockwork"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type mockStorage struct {
	items []*clockwork.Metadata
}

func (m *mockStorage) Store(ctx context.Context, metadata *clockwork.Metadata) error {
	m.items = append(m.items, metadata)
	return nil
}

func (m *mockStorage) Get(ctx context.Context, id string) (*clockwork.Metadata, error) {
	for _, item := range m.items {
		if item.ID == id {
			return item, nil
		}
	}
	return nil, nil
}

func (m *mockStorage) List(ctx context.Context, limit int) ([]*clockwork.Metadata, error) {
	return m.items, nil
}

func (m *mockStorage) Cleanup(ctx context.Context, maxAge time.Duration) error {
	return nil
}

func TestMiddleware_ActivatesOnlyWhenHeaderPresent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := clockwork.Config{Enabled: true, HeaderName: "X-Clockwork", IDHeader: "X-Clockwork-Id"}
	cfg.Normalize()
	store := &mockStorage{}
	cw := clockwork.NewClockwork(cfg, store)

	router := gin.New()
	router.Use(Middleware(cw, nil))
	router.GET("/ok", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	resNoHeader := httptest.NewRecorder()
	reqNoHeader := httptest.NewRequest(http.MethodGet, "/ok", nil)
	router.ServeHTTP(resNoHeader, reqNoHeader)
	require.Equal(t, http.StatusOK, resNoHeader.Code)
	require.Empty(t, resNoHeader.Header().Get("X-Clockwork-Id"))
	require.Len(t, store.items, 0)

	resWithHeader := httptest.NewRecorder()
	reqWithHeader := httptest.NewRequest(http.MethodGet, "/ok", nil)
	reqWithHeader.Header["X-Clockwork"] = []string{""}
	router.ServeHTTP(resWithHeader, reqWithHeader)
	require.Equal(t, http.StatusOK, resWithHeader.Code)
	require.NotEmpty(t, resWithHeader.Header().Get("X-Clockwork-Id"))
	require.Len(t, store.items, 1)
}

func TestMiddleware_SkipsFaviconEvenWhenHeaderPresent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := clockwork.Config{Enabled: true, HeaderName: "X-Clockwork", IDHeader: "X-Clockwork-Id"}
	cfg.Normalize()
	store := &mockStorage{}
	cw := clockwork.NewClockwork(cfg, store)

	router := gin.New()
	router.Use(Middleware(cw, nil))
	router.GET("/favicon.ico", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	req.Header["X-Clockwork"] = []string{""}
	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusNoContent, res.Code)
	require.Empty(t, res.Header().Get("X-Clockwork-Id"))
	require.Len(t, store.items, 0)
}

func TestMiddleware_SkipsFaviconPathVariants(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := clockwork.Config{Enabled: true, HeaderName: "X-Clockwork", IDHeader: "X-Clockwork-Id"}
	cfg.Normalize()
	store := &mockStorage{}
	cw := clockwork.NewClockwork(cfg, store)

	router := gin.New()
	router.Use(Middleware(cw, nil))
	router.GET("/favicon.ico/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	router.GET("/static/favicon.ico", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for _, path := range []string{"/favicon.ico/", "/static/favicon.ico"} {
		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header["X-Clockwork"] = []string{""}
		router.ServeHTTP(res, req)

		require.Equal(t, http.StatusNoContent, res.Code)
		require.Empty(t, res.Header().Get("X-Clockwork-Id"))
		require.Empty(t, res.Header().Get("X-Clockwork-Version"))
	}
	require.Len(t, store.items, 0)
}

func TestMiddleware_StoresTimelineDataWithResponseDuration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := clockwork.Config{Enabled: true, HeaderName: "X-Clockwork", IDHeader: "X-Clockwork-Id"}
	cfg.Normalize()
	store := clockwork.NewInMemoryStorage(20, 1024*1024)
	cw := clockwork.NewClockwork(cfg, store)

	router := gin.New()
	router.Use(Middleware(cw, nil))
	router.GET("/ok", func(c *gin.Context) {
		time.Sleep(2 * time.Millisecond)
		c.Status(http.StatusOK)
	})
	RegisterRoutes(router, cw, nil)

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header["X-Clockwork"] = []string{""}
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code)
	clockworkID := res.Header().Get("X-Clockwork-Id")
	require.NotEmpty(t, clockworkID)

	metadataRes := httptest.NewRecorder()
	metadataReq := httptest.NewRequest(http.MethodGet, "/__clockwork/"+clockworkID, nil)
	router.ServeHTTP(metadataRes, metadataReq)
	require.Equal(t, http.StatusOK, metadataRes.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(metadataRes.Body.Bytes(), &payload))

	timelineRaw, ok := payload["timelineData"]
	require.True(t, ok)
	timeline, ok := timelineRaw.([]any)
	require.True(t, ok)
	require.NotEmpty(t, timeline)

	responseDuration, ok := payload["responseDuration"].(float64)
	require.True(t, ok)
	require.Greater(t, responseDuration, 0.0)

	controller, ok := payload["controller"].(string)
	require.True(t, ok)
	require.NotContains(t, controller, "github.com")
	require.Contains(t, controller, "[/ok]")

	logRaw, ok := payload["log"]
	require.True(t, ok)
	logEntries, ok := logRaw.([]any)
	require.True(t, ok)
	require.NotEmpty(t, logEntries)

	firstLog, ok := logEntries[0].(map[string]any)
	require.True(t, ok)
	_, hasTime := firstLog["time"]
	require.True(t, hasTime)
	traceRaw, hasTrace := firstLog["trace"]
	require.True(t, hasTrace)
	traceFrames, ok := traceRaw.([]any)
	require.True(t, ok)
	require.NotEmpty(t, traceFrames)
}

func TestClockworkRoute_HeaderIDStrictOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := clockwork.Config{Enabled: true, HeaderName: "X-Clockwork", IDHeader: "X-Clockwork-Id"}
	cfg.Normalize()
	store := clockwork.NewInMemoryStorage(10, 1024*1024)
	cw := clockwork.NewClockwork(cfg, store)

	headerMeta := &clockwork.Metadata{ID: "header-id", Method: "GET", URI: "/header"}
	routeMeta := &clockwork.Metadata{ID: "route-id", Method: "GET", URI: "/route"}
	require.NoError(t, cw.SaveMetadata(context.Background(), headerMeta))
	require.NoError(t, cw.SaveMetadata(context.Background(), routeMeta))

	router := gin.New()
	RegisterRoutes(router, cw, nil)

	req := httptest.NewRequest(http.MethodGet, "/__clockwork/route-id", nil)
	req.Header.Set("X-Clockwork-Id", "header-id")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code)

	var got clockwork.Metadata
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &got))
	require.Equal(t, "header-id", got.ID)
}

func TestClockworkRoute_HeaderIDStrictOverrideNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := clockwork.Config{Enabled: true, HeaderName: "X-Clockwork", IDHeader: "X-Clockwork-Id"}
	cfg.Normalize()
	store := clockwork.NewInMemoryStorage(10, 1024*1024)
	cw := clockwork.NewClockwork(cfg, store)

	require.NoError(t, cw.SaveMetadata(context.Background(), &clockwork.Metadata{ID: "route-id", Method: "GET", URI: "/route"}))

	router := gin.New()
	RegisterRoutes(router, cw, nil)

	req := httptest.NewRequest(http.MethodGet, "/__clockwork/route-id", nil)
	req.Header.Set("X-Clockwork-Id", "missing-id")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusNotFound, res.Code)
}
