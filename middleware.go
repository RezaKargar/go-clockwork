package clockwork

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Middleware returns a Gin middleware for Clockwork request profiling.
func Middleware(cw *Clockwork, logger *zap.Logger) gin.HandlerFunc {
	if cw == nil || !cw.IsEnabled() {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		if shouldSkipClockworkRequest(c.Request) {
			c.Next()
			c.Writer.Header().Del(cw.config.IDHeader)
			c.Writer.Header().Del("X-Clockwork-Version")
			return
		}

		if !headerExists(c.Request.Header, cw.config.HeaderName) {
			c.Next()
			return
		}

		collector := cw.NewCollector(c.Request.Method, c.Request.RequestURI)
		if collector == nil {
			c.Next()
			return
		}

		collector.SetHeaders(extractSafeHeaders(c.Request.Header))
		collector.SetURL(buildRequestURL(c.Request))
		collector.AddLogEntry("info", "clockwork capture enabled", map[string]interface{}{
			"method": c.Request.Method,
			"uri":    c.Request.RequestURI,
		})

		traceID, spanID := traceFromContext(c.Request.Context())
		collector.SetTrace(traceID, spanID)
		if traceID != "" {
			cw.RegisterTrace(traceID, collector)
		}

		ctx := ContextWithCollector(c.Request.Context(), collector)
		c.Request = c.Request.WithContext(ctx)

		c.Header(cw.config.IDHeader, collector.ID())
		c.Header("X-Clockwork-Version", ProtocolVersion)

		start := time.Now()
		c.Next()

		duration := time.Since(start)
		if controller := resolveControllerName(c); controller != "" {
			collector.SetController(controller)
		}
		collector.AddLogEntry("info", "request completed", map[string]interface{}{
			"status":      c.Writer.Status(),
			"duration_ms": durationMs(duration),
		})
		if err := cw.CompleteRequest(c.Request.Context(), collector, c.Writer.Status(), duration); err != nil && logger != nil {
			logger.Warn("failed to persist clockwork metadata",
				zap.String("id", collector.ID()),
				zap.Error(err),
			)
		}
	}
}

func shouldSkipClockworkRequest(req *http.Request) bool {
	if req == nil || req.URL == nil {
		return false
	}

	path := strings.TrimSpace(req.URL.Path)
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		path = "/"
	}

	if strings.EqualFold(path, "/favicon.ico") || strings.HasSuffix(strings.ToLower(path), "/favicon.ico") {
		return true
	}
	if strings.HasPrefix(path, "/__clockwork") {
		return true
	}
	return false
}

func buildRequestURL(req *http.Request) string {
	if req == nil || req.URL == nil {
		return ""
	}
	if req.URL.IsAbs() {
		return req.URL.String()
	}

	scheme := req.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if req.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	host := req.Host
	if host == "" {
		return req.URL.String()
	}
	return scheme + "://" + host + req.URL.RequestURI()
}

func resolveControllerName(c *gin.Context) string {
	if c == nil {
		return ""
	}
	handler := compactHandlerName(strings.TrimSpace(c.HandlerName()))
	route := strings.TrimSpace(c.FullPath())
	switch {
	case handler != "" && route != "":
		return handler + " [" + route + "]"
	case handler != "":
		return handler
	case route != "":
		return route
	default:
		return ""
	}
}

func compactHandlerName(full string) string {
	if full == "" {
		return ""
	}

	trimmed := strings.TrimSuffix(full, "-fm")
	if slash := strings.LastIndex(trimmed, "/"); slash >= 0 && slash < len(trimmed)-1 {
		trimmed = trimmed[slash+1:]
	}

	trimmed = strings.ReplaceAll(trimmed, "(*", "")
	trimmed = strings.ReplaceAll(trimmed, ")", "")
	trimmed = strings.ReplaceAll(trimmed, "(.", ".")

	parts := strings.Split(trimmed, ".")
	switch {
	case len(parts) >= 3:
		return parts[0] + "." + parts[len(parts)-2] + "." + parts[len(parts)-1]
	case len(parts) >= 2:
		return parts[len(parts)-2] + "." + parts[len(parts)-1]
	default:
		return trimmed
	}
}

func traceFromContext(ctx context.Context) (traceID, spanID string) {
	span := trace.SpanFromContext(ctx)
	spanCtx := span.SpanContext()
	if !spanCtx.IsValid() {
		return "", ""
	}
	return spanCtx.TraceID().String(), spanCtx.SpanID().String()
}

func headerExists(headers http.Header, headerName string) bool {
	if headerName == "" {
		return false
	}
	_, ok := headers[http.CanonicalHeaderKey(headerName)]
	return ok
}

// extractSafeHeaders extracts a safe subset of headers to avoid sensitive values.
func extractSafeHeaders(headers http.Header) map[string]string {
	safeHeaders := make(map[string]string)
	safeHeaderNames := map[string]bool{
		"Content-Type":    true,
		"Accept":          true,
		"User-Agent":      true,
		"Accept-Language": true,
		"Accept-Encoding": true,
		"X-Request-ID":    true,
		"X-City-ID":       true,
		"X-Stage":         true,
		"X-App-Version":   true,
		"Origin":          true,
		"Referer":         true,
	}

	for key, values := range headers {
		if safeHeaderNames[key] && len(values) > 0 {
			safeHeaders[key] = values[0]
		}
	}

	return safeHeaders
}
