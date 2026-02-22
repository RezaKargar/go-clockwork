package e2e

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RezaKargar/go-clockwork"
	clockworkhttp "github.com/RezaKargar/go-clockwork/middleware/http"
	"github.com/stretchr/testify/require"
)

func TestE2E_InMemory_NetHTTPFlow(t *testing.T) {
	cfg := clockwork.DefaultConfig()
	cfg.Normalize()

	store := clockwork.NewInMemoryStorage(cfg.MaxRequests, cfg.MaxStorageBytes)
	cw := clockwork.NewClockwork(cfg, store)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	clockworkhttp.RegisterMetadataRoute(mux, cw)
	handler := clockworkhttp.Middleware(cw, mux)

	captureRes := httptest.NewRecorder()
	captureReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	captureReq.Header.Set(cfg.HeaderName, "")
	handler.ServeHTTP(captureRes, captureReq)
	require.Equal(t, http.StatusOK, captureRes.Code)

	id := captureRes.Header().Get(cfg.IDHeader)
	require.NotEmpty(t, id)

	metaRes := httptest.NewRecorder()
	metaReq := httptest.NewRequest(http.MethodGet, "/__clockwork/"+id, nil)
	handler.ServeHTTP(metaRes, metaReq)
	require.Equal(t, http.StatusOK, metaRes.Code)
}
