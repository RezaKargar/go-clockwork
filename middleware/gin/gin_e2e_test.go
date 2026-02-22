package gin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RezaKargar/go-clockwork"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestE2E_InMemory_GinFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := clockwork.DefaultConfig()
	cfg.Normalize()

	store := clockwork.NewInMemoryStorage(cfg.MaxRequests, cfg.MaxStorageBytes)
	cw := clockwork.NewClockwork(cfg, store)

	router := gin.New()
	router.Use(Middleware(cw, nil))
	router.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })
	RegisterRoutes(router, cw, nil)

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
