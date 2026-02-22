package memcache

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/RezaKargar/go-clockwork"
	clockworkhttp "github.com/RezaKargar/go-clockwork/middleware/http"
	"github.com/stretchr/testify/require"
)

func TestE2E_Memcache_Flow(t *testing.T) {
	endpoint := os.Getenv("CLOCKWORK_MEMCACHE_ENDPOINT")
	if endpoint == "" {
		t.Skip("CLOCKWORK_MEMCACHE_ENDPOINT is not set")
	}
	store, err := New(Config{
		Endpoints:  []string{endpoint},
		TTL:        time.Hour,
		MaxEntries: 200,
	})
	require.NoError(t, err)
	runStorageE2E(t, store)
}

func runStorageE2E(t *testing.T, store clockwork.Storage) {
	t.Helper()

	cfg := clockwork.DefaultConfig()
	cfg.Normalize()
	cw := clockwork.NewClockwork(cfg, store)

	mux := http.NewServeMux()
	mux.HandleFunc("/items", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusAccepted) })
	clockworkhttp.RegisterMetadataRoute(mux, cw)
	handler := clockworkhttp.Middleware(cw, mux)

	captureRes := httptest.NewRecorder()
	captureReq := httptest.NewRequest(http.MethodGet, "/items", nil)
	captureReq.Header.Set(cfg.HeaderName, "")
	handler.ServeHTTP(captureRes, captureReq)
	require.Equal(t, http.StatusAccepted, captureRes.Code)

	id := captureRes.Header().Get(cfg.IDHeader)
	require.NotEmpty(t, id)

	metaRes := httptest.NewRecorder()
	metaReq := httptest.NewRequest(http.MethodGet, "/__clockwork/"+id, nil)
	handler.ServeHTTP(metaRes, metaReq)
	require.Equal(t, http.StatusOK, metaRes.Code)
}
