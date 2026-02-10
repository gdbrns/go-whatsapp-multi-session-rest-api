package router

import (
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
			return c.Method() != fiber.MethodGet
		},
		Expiration: time.Duration(ttl) * time.Second,
	})
}
