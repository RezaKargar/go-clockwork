package clockwork

import (
	"context"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// ShouldSkipPath returns true if the request path should not be profiled (e.g. favicon, __clockwork routes).
func ShouldSkipPath(path string) bool {
	path = strings.TrimSpace(strings.TrimSuffix(path, "/"))
	if path == "" {
		path = "/"
	}
	if strings.EqualFold(path, "/favicon.ico") || strings.HasSuffix(strings.ToLower(path), "/favicon.ico") {
		return true
	}
	return strings.HasPrefix(path, "/__clockwork")
}

// ShouldCapture returns true if the request headers indicate Clockwork capture is requested (e.g. X-Clockwork header present).
func ShouldCapture(headers http.Header, headerName string) bool {
	if headerName == "" {
		return false
	}
	_, ok := headers[http.CanonicalHeaderKey(headerName)]
	return ok
}

// BuildRequestURL reconstructs the full request URL from the request.
func BuildRequestURL(req *http.Request) string {
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

// ExtractSafeHeaders returns a safe subset of headers to avoid sensitive values.
func ExtractSafeHeaders(headers http.Header) map[string]string {
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

// TraceFromContext returns trace and span IDs from the context (OpenTelemetry).
func TraceFromContext(ctx context.Context) (traceID, spanID string) {
	span := trace.SpanFromContext(ctx)
	spanCtx := span.SpanContext()
	if !spanCtx.IsValid() {
		return "", ""
	}
	return spanCtx.TraceID().String(), spanCtx.SpanID().String()
}

// NewRequestCapture decides whether to capture this request and, if so, returns a new Collector.
// path is used for skip logic (e.g. favicon, /__clockwork); uri is stored on the collector (e.g. request URI).
// Framework middleware should call this first; if ok is false, skip Clockwork and run the next handler.
func NewRequestCapture(cw *Clockwork, method, path, uri string, headers http.Header) (*Collector, bool) {
	if cw == nil || !cw.IsEnabled() {
		return nil, false
	}
	if ShouldSkipPath(path) || !ShouldCapture(headers, cw.Config().HeaderName) {
		return nil, false
	}
	collector := cw.NewCollector(method, uri)
	if collector == nil {
		return nil, false
	}
	return collector, true
}
