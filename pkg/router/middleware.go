package router

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func HttpRealIP() fiber.Handler {
	return func(c *fiber.Ctx) error {
		xForwardedFor := c.Get(http.CanonicalHeaderKey("X-Forwarded-For"))
		if xForwardedFor != "" {
			parts := strings.Split(xForwardedFor, ",")
			if len(parts) > 0 {
				c.Locals("remote_ip", strings.TrimSpace(parts[0]))
			}
		} else {
			xRealIP := c.Get(http.CanonicalHeaderKey("X-Real-IP"))
			if xRealIP != "" {
				c.Locals("remote_ip", strings.TrimSpace(xRealIP))
			}
		}
		return c.Next()
	}
}

// HttpRequestID injects a request ID into context and response headers.
// It preserves an incoming X-Request-ID when present; otherwise it generates
// a UUIDv4. The ID is stored in Locals("request_id") for logging.
func HttpRequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqID := strings.TrimSpace(c.Get("X-Request-ID"))
		if reqID == "" {
			reqID = uuid.NewString()
		}
		c.Locals("request_id", reqID)
		c.Set("X-Request-ID", reqID)
		return c.Next()
	}
}
