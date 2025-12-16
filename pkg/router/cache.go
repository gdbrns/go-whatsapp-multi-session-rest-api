package router

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
)

func HttpCacheInMemory(capacity int, ttl int) fiber.Handler {
	if capacity <= 0 {
		capacity = 1000
	}
	if ttl <= 0 {
		ttl = 5
	}
	_ = capacity // capacity reserved for future use with custom storage
	return cache.New(cache.Config{
		Expiration: time.Duration(ttl) * time.Second,
	})
}
