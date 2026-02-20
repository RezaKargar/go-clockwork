package gin

import (
	"github.com/RezaKargar/go-clockwork"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Middleware returns Gin middleware for Clockwork request profiling.
func Middleware(cw *clockwork.Clockwork, logger *zap.Logger) gin.HandlerFunc {
	return clockwork.Middleware(cw, logger)
}

// RegisterRoutes registers minimal Clockwork API routes under /__clockwork.
func RegisterRoutes(router *gin.Engine, cw *clockwork.Clockwork, logger *zap.Logger, routeMiddlewares ...gin.HandlerFunc) {
	clockwork.RegisterRoutes(router, cw, logger, routeMiddlewares...)
}
