package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/RezaKargar/go-clockwork"
)

// Middleware returns a net/http middleware for Clockwork request profiling.
func Middleware(cw *clockwork.Clockwork, next http.Handler) http.Handler {
	if cw == nil || !cw.IsEnabled() {
		return next
	}
	if next == nil {
		next = http.NotFoundHandler()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil {
			next.ServeHTTP(w, r)
			return
		}

		collector, ok := clockwork.NewRequestCapture(cw, r.Method, r.URL.Path, r.RequestURI, r.Header)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		collector.SetHeaders(clockwork.ExtractSafeHeaders(r.Header))
		collector.SetURL(clockwork.BuildRequestURL(r))

		traceID, spanID := clockwork.TraceFromContext(r.Context())
		collector.SetTrace(traceID, spanID)
		if traceID != "" {
			cw.RegisterTrace(traceID, collector)
		}

		r = r.WithContext(clockwork.ContextWithCollector(r.Context(), collector))
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		rw.Header().Set(cw.Config().IDHeader, collector.ID())
		rw.Header().Set("X-Clockwork-Version", clockwork.ProtocolVersion)

		started := time.Now()
		next.ServeHTTP(rw, r)
		duration := time.Since(started)

		_ = cw.CompleteRequest(r.Context(), collector, rw.statusCode, duration)
	})
}

// MetadataHandler handles GET /__clockwork/:id lookups.
func MetadataHandler(cw *clockwork.Clockwork) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cw == nil || !cw.IsEnabled() {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id := resolveMetadataID(r, cw.Config().IDHeader)
		if id == "" {
			writeJSONError(w, http.StatusBadRequest, "metadata id is required")
			return
		}

		metadata, err := cw.GetMetadata(r.Context(), id)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "metadata not found")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Clockwork-Version", clockwork.ProtocolVersion)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(metadata)
	})
}

// RegisterMetadataRoute registers GET /__clockwork/:id on provided mux.
func RegisterMetadataRoute(mux *http.ServeMux, cw *clockwork.Clockwork) {
	if mux == nil || cw == nil || !cw.IsEnabled() {
		return
	}
	h := MetadataHandler(cw)
	mux.Handle("GET /__clockwork/{id}", h)
	mux.Handle("GET /__clockwork/", h)
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func resolveMetadataID(r *http.Request, idHeader string) string {
	if r == nil {
		return ""
	}
	if idHeader != "" {
		if headerID := strings.TrimSpace(r.Header.Get(idHeader)); headerID != "" {
			return headerID
		}
	}
	path := strings.TrimSpace(strings.TrimSuffix(r.URL.Path, "/"))
	if path == "" {
		return ""
	}
	segments := strings.Split(path, "/")
	if len(segments) < 3 || segments[1] != "__clockwork" {
		return ""
	}
	return strings.TrimSpace(segments[2])
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Clockwork-Version", clockwork.ProtocolVersion)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
