package fiber

import (
	"net/http"
	"strings"
	"time"

	"github.com/RezaKargar/go-clockwork"
	"github.com/gofiber/fiber/v2"
)

// Middleware returns Fiber middleware for Clockwork request profiling.
func Middleware(cw *clockwork.Clockwork) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if cw == nil || !cw.IsEnabled() {
			return c.Next()
		}

		path := string(c.Path())
		headers := requestHeadersToHTTP(c)
		if clockwork.ShouldSkipPath(path) || !clockwork.ShouldCapture(headers, cw.Config().HeaderName) {
			return c.Next()
		}

		collector := cw.NewCollector(c.Method(), c.Path())
		if collector == nil {
			return c.Next()
		}

		collector.SetHeaders(clockwork.ExtractSafeHeaders(headers))
		collector.SetURL(buildRequestURL(c))

		traceID, spanID := clockwork.TraceFromContext(c.UserContext())
		collector.SetTrace(traceID, spanID)
		if traceID != "" {
			cw.RegisterTrace(traceID, collector)
		}

		c.SetUserContext(clockwork.ContextWithCollector(c.UserContext(), collector))
		c.Set(cw.Config().IDHeader, collector.ID())
		c.Set("X-Clockwork-Version", clockwork.ProtocolVersion)

		started := time.Now()
		err := c.Next()
		duration := time.Since(started)

		status := c.Response().StatusCode()
		if status == 0 {
			status = fiber.StatusOK
		}
		if routePattern := strings.TrimSpace(c.Route().Path); routePattern != "" {
			collector.SetController(routePattern)
		}

		_ = cw.CompleteRequest(c.UserContext(), collector, status, duration)
		return err
	}
}

// RegisterRoutes registers GET /__clockwork/:id on the Fiber app.
func RegisterRoutes(app *fiber.App, cw *clockwork.Clockwork) {
	if app == nil || cw == nil || !cw.IsEnabled() {
		return
	}
	app.Get("/__clockwork/:id", func(c *fiber.Ctx) error {
		id := resolveMetadataID(c, cw.Config().IDHeader)
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "metadata id is required"})
		}

		metadata, err := cw.GetMetadata(c.UserContext(), id)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "metadata not found"})
		}

		c.Set("X-Clockwork-Version", clockwork.ProtocolVersion)
		c.Set("Content-Type", "application/json")
		return c.Status(fiber.StatusOK).JSON(metadata)
	})
}

func requestHeadersToHTTP(c *fiber.Ctx) http.Header {
	h := make(http.Header)
	c.Request().Header.VisitAll(func(key, value []byte) {
		h.Add(string(key), string(value))
	})
	return h
}

func buildRequestURL(c *fiber.Ctx) string {
	scheme := c.Get("X-Forwarded-Proto")
	if scheme == "" {
		if c.Protocol() == "https" {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := c.Hostname()
	if host == "" {
		return ""
	}
	return scheme + "://" + host + c.OriginalURL()
}

func resolveMetadataID(c *fiber.Ctx, idHeader string) string {
	if idHeader != "" {
		if headerID := strings.TrimSpace(c.Get(idHeader)); headerID != "" {
			return headerID
		}
	}
	return strings.TrimSpace(c.Params("id"))
}
