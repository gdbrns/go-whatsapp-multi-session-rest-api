package index

import (
	"github.com/gofiber/fiber/v2"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/router"
)

// Index
// @Summary     Show The Status of The Server
// @Description Get The Server Status
// @Tags        Root
// @Produce     json
// @Success     200
// @Router      / [get]
func Index(c *fiber.Ctx) error {
	return router.ResponseSuccess(c, "Go WhatsApp Multi-Device REST is running")
}
