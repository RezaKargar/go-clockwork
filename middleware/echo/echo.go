package echo

import (
	"net/http"
	"strings"
	"time"

	"github.com/RezaKargar/go-clockwork"
	"github.com/labstack/echo/v4"
)

// Middleware returns Echo middleware for Clockwork request profiling.
func Middleware(cw *clockwork.Clockwork) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if cw == nil || !cw.IsEnabled() {
				return next(c)
			}

			req := c.Request()
			if req == nil {
				return next(c)
			}

			if clockwork.ShouldSkipPath(req.URL.Path) || !clockwork.ShouldCapture(req.Header, cw.Config().HeaderName) {
				return next(c)
			}

			collector := cw.NewCollector(req.Method, req.RequestURI)
			if collector == nil {
				return next(c)
			}

			collector.SetHeaders(clockwork.ExtractSafeHeaders(req.Header))
			collector.SetURL(clockwork.BuildRequestURL(req))

			traceID, spanID := clockwork.TraceFromContext(req.Context())
			collector.SetTrace(traceID, spanID)
			if traceID != "" {
				cw.RegisterTrace(traceID, collector)
			}

			c.SetRequest(req.WithContext(clockwork.ContextWithCollector(req.Context(), collector)))
			c.Response().Header().Set(cw.Config().IDHeader, collector.ID())
			c.Response().Header().Set("X-Clockwork-Version", clockwork.ProtocolVersion)

			started := time.Now()
			err := next(c)
			duration := time.Since(started)

			status := c.Response().Status
			if status == 0 {
				status = http.StatusOK
			}
			if route := c.Path(); strings.TrimSpace(route) != "" {
				collector.SetController(route)
			}

			_ = cw.CompleteRequest(c.Request().Context(), collector, status, duration)
			return err
		}
	}
}

// RegisterRoutes registers GET /__clockwork/:id on the Echo instance.
func RegisterRoutes(e *echo.Echo, cw *clockwork.Clockwork) {
	if e == nil || cw == nil || !cw.IsEnabled() {
		return
	}
	e.GET("/__clockwork/:id", func(c echo.Context) error {
		id := resolveMetadataID(c, cw.Config().IDHeader)
		if id == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "metadata id is required"})
		}

		metadata, err := cw.GetMetadata(c.Request().Context(), id)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "metadata not found"})
		}

		c.Response().Header().Set("X-Clockwork-Version", clockwork.ProtocolVersion)
		return c.JSON(http.StatusOK, metadata)
	})
}

func resolveMetadataID(c echo.Context, idHeader string) string {
	if idHeader != "" {
		if headerID := strings.TrimSpace(c.Request().Header.Get(idHeader)); headerID != "" {
			return headerID
		}
	}
	return strings.TrimSpace(c.Param("id"))
}
