package presence

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/router"
	pkgWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/whatsapp"
	typWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/types"
)

// getDeviceContext extracts device context from auth middleware
func getDeviceContext(c *fiber.Ctx) (deviceID string, jid string) {
	deviceID = c.Locals("device_id").(string)
	jidVal := c.Locals("device_jid")
	if jidVal != nil {
		jid = jidVal.(string)
	}
	return
}

func parseTimer(timer string) (int, error) {
	switch timer {
	case "off":
		return 0, nil
	case "24h":
		return 86400, nil
	case "7d":
		return 604800, nil
	case "90d":
		return 7776000, nil
	default:
		return 0, fmt.Errorf("invalid timer value")
	}
}

func SendChatPresence(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	chatJID := c.Params("chat_jid")

	var reqPresence typWhatsApp.RequestPresence
	err := c.BodyParser(&reqPresence)
	if err != nil {
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	phoneJID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, chatJID)

	isComposing := reqPresence.State == "composing"
	isAudio := reqPresence.Media == "audio"

	pkgWhatsApp.WhatsAppComposeStatus(ctx, jid, deviceID, phoneJID, isComposing, isAudio)

	return router.ResponseSuccess(c, "Success send presence")
}

func UpdateStatus(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	var req struct {
		Status string `json:"status"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	isAvailable := req.Status == "available"
	pkgWhatsApp.WhatsAppPresence(ctx, jid, deviceID, isAvailable)

	return router.ResponseSuccess(c, "Success update presence status")
}

func SetDisappearingTimer(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	chatJID := c.Params("chat_jid")

	var req struct {
		Timer string `json:"timer"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	timer, err := parseTimer(req.Timer)
	if err != nil {
		return router.ResponseBadRequest(c, "Invalid timer value")
	}

	err = pkgWhatsApp.WhatsAppSetDisappearingTimer(ctx, jid, deviceID, timer, chatJID)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccess(c, "Success set disappearing timer")
}
