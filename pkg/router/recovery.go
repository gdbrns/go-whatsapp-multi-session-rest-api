package router

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/log"
)

// RecoveryMiddleware converts panics into structured JSON responses and logs them.
// It must be registered before application routes.
func RecoveryMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if rec := recover(); rec != nil {
				message := fmt.Sprintf("%v", rec)
				resp := Response{
					Status:  false,
					Code:    fiber.StatusInternalServerError,
					Message: message,
				}
				// Preserve legacy error field for compatibility
				resp.Error = message
				log.Print(c).WithField("request_id", c.Locals("request_id")).Error("panic recovered: " + message)
				_ = c.Status(resp.Code).JSON(resp)
			}
		}()
		return c.Next()
	}
}

