package e2e

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/RezaKargar/go-clockwork"
	ginmw "github.com/RezaKargar/go-clockwork/middleware/gin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestE2E_InMemory_GinFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := clockwork.DefaultConfig()
	cfg.StorageType = "memory"
	cfg.Normalize()

	store, err := clockwork.NewStorage(cfg)
	require.NoError(t, err)
	cw := clockwork.NewClockwork(cfg, store)

	router := gin.New()
	router.Use(ginmw.Middleware(cw, nil))
	router.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })
	ginmw.RegisterRoutes(router, cw, nil)

	captureRes := httptest.NewRecorder()
	captureReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	captureReq.Header.Set(cfg.HeaderName, "")
	router.ServeHTTP(captureRes, captureReq)
	require.Equal(t, http.StatusOK, captureRes.Code)

	id := captureRes.Header().Get(cfg.IDHeader)
	require.NotEmpty(t, id)

	metaRes := httptest.NewRecorder()
	metaReq := httptest.NewRequest(http.MethodGet, "/__clockwork/"+id, nil)
	router.ServeHTTP(metaRes, metaReq)
	require.Equal(t, http.StatusOK, metaRes.Code)
}

func TestE2E_Redis_GinFlow(t *testing.T) {
	endpoint := os.Getenv("CLOCKWORK_REDIS_ENDPOINT")
	if endpoint == "" {
		t.Skip("CLOCKWORK_REDIS_ENDPOINT is not set")
	}
	runStorageFlow(t, clockwork.Config{
		Enabled:       true,
		HeaderName:    "X-Clockwork",
		IDHeader:      "X-Clockwork-Id",
		StorageType:   "redis",
		RedisEndpoint: endpoint,
	})
}

func TestE2E_Memcache_GinFlow(t *testing.T) {
	endpoint := os.Getenv("CLOCKWORK_MEMCACHE_ENDPOINT")
	if endpoint == "" {
		t.Skip("CLOCKWORK_MEMCACHE_ENDPOINT is not set")
	}
	runStorageFlow(t, clockwork.Config{
		Enabled:           true,
		HeaderName:        "X-Clockwork",
		IDHeader:          "X-Clockwork-Id",
		StorageType:       "memcache",
		MemcacheEndpoints: []string{endpoint},
	})
}

func runStorageFlow(t *testing.T, cfg clockwork.Config) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg.Normalize()

	store, err := clockwork.NewStorage(cfg)
	require.NoError(t, err)
	cw := clockwork.NewClockwork(cfg, store)

	router := gin.New()
	router.Use(ginmw.Middleware(cw, nil))
	router.GET("/items", func(c *gin.Context) { c.Status(http.StatusAccepted) })
	ginmw.RegisterRoutes(router, cw, nil)

	captureRes := httptest.NewRecorder()
	captureReq := httptest.NewRequest(http.MethodGet, "/items", nil)
	captureReq.Header.Set(cfg.HeaderName, "")
	router.ServeHTTP(captureRes, captureReq)
	require.Equal(t, http.StatusAccepted, captureRes.Code)

	id := captureRes.Header().Get(cfg.IDHeader)
	require.NotEmpty(t, id)

	metaRes := httptest.NewRecorder()
	metaReq := httptest.NewRequest(http.MethodGet, "/__clockwork/"+id, nil)
	router.ServeHTTP(metaRes, metaReq)
	require.Equal(t, http.StatusOK, metaRes.Code)
}
