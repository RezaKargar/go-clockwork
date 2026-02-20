package clockwork

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RegisterRoutes registers Clockwork API routes under /__clockwork.
// Optional routeMiddlewares can be used to mark routes anonymous in host applications.
func RegisterRoutes(router *gin.Engine, cw *Clockwork, logger *zap.Logger, routeMiddlewares ...gin.HandlerFunc) {
	if cw == nil || !cw.IsEnabled() {
		return
	}

	group := router.Group("/__clockwork", routeMiddlewares...)

	group.GET("/:id", func(c *gin.Context) {
		id := resolveMetadataID(c, cw.config.IDHeader)
		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "metadata id is required"})
			return
		}

		metadata, err := cw.GetMetadata(c.Request.Context(), id)
		if err != nil {
			if logger != nil {
				logger.Warn("clockwork metadata not found", zap.String("id", id), zap.Error(err))
			}
			c.JSON(http.StatusNotFound, gin.H{"error": "metadata not found"})
			return
		}

		setClockworkResponseHeaders(c)
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

func setClockworkResponseHeaders(c *gin.Context) {
	c.Header("X-Clockwork-Version", ProtocolVersion)
	c.Header("Content-Type", "application/json")
}
