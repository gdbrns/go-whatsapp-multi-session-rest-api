package router

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
)

func HttpCacheInMemory(ttl int) fiber.Handler {
	if ttl <= 0 {
		ttl = 5
	}
	return cache.New(cache.Config{
		Next: func(c *fiber.Ctx) bool {
			// Only cache GET requests
			if c.Method() != fiber.MethodGet {
				return true
			}
			// Skip caching for dynamic/status endpoints that should always be fresh
			p := c.Path()
			if strings.HasSuffix(p, "/status") ||
				strings.HasSuffix(p, "/health") ||
				strings.Contains(p, "/qr") ||
				strings.Contains(p, "docs") {
				return true
			}
			return false
		},
		Expiration: time.Duration(ttl) * time.Second,
	})
}
