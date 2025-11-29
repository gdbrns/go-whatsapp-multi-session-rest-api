package router

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
)

func HttpCacheInMemory(cap int, ttl int) fiber.Handler {
	if cap <= 0 {
		cap = 1000
	}
	if ttl <= 0 {
		ttl = 5
	}
	return cache.New(cache.Config{
		Expiration: time.Duration(ttl) * time.Second,
	})
}
