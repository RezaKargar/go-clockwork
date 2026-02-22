package chi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/RezaKargar/go-clockwork"
	chimw "github.com/go-chi/chi/v5"
)

// Middleware returns Chi middleware for Clockwork request profiling.
func Middleware(cw *clockwork.Clockwork) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
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

			if clockwork.ShouldSkipPath(r.URL.Path) || !clockwork.ShouldCapture(r.Header, cw.Config().HeaderName) {
				next.ServeHTTP(w, r)
				return
			}

			collector := cw.NewCollector(r.Method, r.RequestURI)
			if collector == nil {
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

			if routePattern := resolveControllerName(r); routePattern != "" {
				collector.SetController(routePattern)
			}

			_ = cw.CompleteRequest(r.Context(), collector, rw.statusCode, duration)
		})
	}
}

// RegisterRoutes registers GET /__clockwork/:id on the Chi router.
func RegisterRoutes(r chimw.Router, cw *clockwork.Clockwork) {
	if r == nil || cw == nil || !cw.IsEnabled() {
		return
	}
	r.Get("/__clockwork/{id}", MetadataHandler(cw).ServeHTTP)
}

// MetadataHandler returns an http.Handler for GET /__clockwork/:id.
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

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func resolveControllerName(r *http.Request) string {
	if r == nil {
		return ""
	}
	ctx := chimw.RouteContext(r.Context())
	if ctx == nil {
		return ""
	}
	pattern := strings.TrimSpace(ctx.RoutePattern())
	if pattern == "" {
		return ""
	}
	return pattern
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
	ctx := chimw.RouteContext(r.Context())
	if ctx != nil {
		if id := strings.TrimSpace(ctx.URLParam("id")); id != "" {
			return id
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
