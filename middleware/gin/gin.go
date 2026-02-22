package gin

import (
	"net/http"
	"strings"
	"time"

	"github.com/RezaKargar/go-clockwork"
	"github.com/gin-gonic/gin"
)

// Middleware returns Gin middleware for Clockwork request profiling.
func Middleware(cw *clockwork.Clockwork, logger clockwork.Logger) gin.HandlerFunc {
	if cw == nil || !cw.IsEnabled() {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		if c.Request != nil && clockwork.ShouldSkipPath(c.Request.URL.Path) {
			c.Next()
			c.Writer.Header().Del(cw.Config().IDHeader)
			c.Writer.Header().Del("X-Clockwork-Version")
			return
		}

		if c.Request == nil || !clockwork.ShouldCapture(c.Request.Header, cw.Config().HeaderName) {
			c.Next()
			return
		}

		collector := cw.NewCollector(c.Request.Method, c.Request.RequestURI)
		if collector == nil {
			c.Next()
			return
		}

		collector.SetHeaders(clockwork.ExtractSafeHeaders(c.Request.Header))
		collector.SetURL(clockwork.BuildRequestURL(c.Request))
		collector.AddLogEntry("info", "clockwork capture enabled", map[string]interface{}{
			"method": c.Request.Method,
			"uri":    c.Request.RequestURI,
		})

		traceID, spanID := clockwork.TraceFromContext(c.Request.Context())
		collector.SetTrace(traceID, spanID)
		if traceID != "" {
			cw.RegisterTrace(traceID, collector)
		}

		ctx := clockwork.ContextWithCollector(c.Request.Context(), collector)
		c.Request = c.Request.WithContext(ctx)

		c.Header(cw.Config().IDHeader, collector.ID())
		c.Header("X-Clockwork-Version", clockwork.ProtocolVersion)

		start := time.Now()
		c.Next()

		duration := time.Since(start)
		if controller := resolveControllerName(c); controller != "" {
			collector.SetController(controller)
		}
		collector.AddLogEntry("info", "request completed", map[string]interface{}{
			"status":      c.Writer.Status(),
			"duration_ms": duration.Milliseconds(),
		})
		if err := cw.CompleteRequest(c.Request.Context(), collector, c.Writer.Status(), duration); err != nil && logger != nil {
			logger.Warn("failed to persist clockwork metadata", "id", collector.ID(), "error", err)
		}
	}
}

// RegisterRoutes registers Clockwork API routes under /__clockwork.
func RegisterRoutes(router *gin.Engine, cw *clockwork.Clockwork, logger clockwork.Logger, routeMiddlewares ...gin.HandlerFunc) {
	if cw == nil || !cw.IsEnabled() {
		return
	}

	group := router.Group("/__clockwork", routeMiddlewares...)

	group.GET("/:id", func(c *gin.Context) {
		id := resolveMetadataID(c, cw.Config().IDHeader)
		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "metadata id is required"})
			return
		}

		metadata, err := cw.GetMetadata(c.Request.Context(), id)
		if err != nil {
			if logger != nil {
				logger.Warn("clockwork metadata not found", "id", id, "error", err)
			}
			c.JSON(http.StatusNotFound, gin.H{"error": "metadata not found"})
			return
		}

		c.Header("X-Clockwork-Version", clockwork.ProtocolVersion)
		c.Header("Content-Type", "application/json")
		c.JSON(http.StatusOK, metadata)
	})
}

func resolveMetadataID(c *gin.Context, idHeaderName string) string {
	if idHeaderName != "" {
		if headerID := strings.TrimSpace(c.GetHeader(idHeaderName)); headerID != "" {
			return headerID
		}
	}
	return strings.TrimSpace(c.Param("id"))
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
