package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RezaKargar/go-clockwork"
	"github.com/stretchr/testify/require"
)

func TestMiddleware_ActivatesOnlyWhenHeaderPresent(t *testing.T) {
	cfg := clockwork.DefaultConfig()
	cfg.Normalize()
	store := clockwork.NewInMemoryStorage(20, 1024*1024)
	cw := clockwork.NewClockwork(cfg, store)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := Middleware(cw, next)

	without := httptest.NewRecorder()
	withoutReq := httptest.NewRequest(http.MethodGet, "/ok", nil)
	handler.ServeHTTP(without, withoutReq)
	require.Empty(t, without.Header().Get(cfg.IDHeader))

	withHeader := httptest.NewRecorder()
	withReq := httptest.NewRequest(http.MethodGet, "/ok", nil)
	withReq.Header.Set(cfg.HeaderName, "")
	handler.ServeHTTP(withHeader, withReq)
	require.NotEmpty(t, withHeader.Header().Get(cfg.IDHeader))
	require.Equal(t, clockwork.ProtocolVersion, withHeader.Header().Get("X-Clockwork-Version"))
}

func TestMetadataHandler_ReturnsCapturedMetadata(t *testing.T) {
	cfg := clockwork.DefaultConfig()
	cfg.Normalize()
	store := clockwork.NewInMemoryStorage(20, 1024*1024)
	cw := clockwork.NewClockwork(cfg, store)

	app := Middleware(cw, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	captureRes := httptest.NewRecorder()
	captureReq := httptest.NewRequest(http.MethodGet, "/ok", nil)
	captureReq.Header.Set(cfg.HeaderName, "")
	app.ServeHTTP(captureRes, captureReq)
	clockworkID := captureRes.Header().Get(cfg.IDHeader)
	require.NotEmpty(t, clockworkID)

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/__clockwork/"+clockworkID, nil)
	MetadataHandler(cw).ServeHTTP(res, req)
	require.Equal(t, http.StatusOK, res.Code)

	var meta clockwork.Metadata
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &meta))
	require.Equal(t, clockworkID, meta.ID)
}
