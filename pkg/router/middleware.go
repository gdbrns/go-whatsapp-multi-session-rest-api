package router

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
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
