package history

import (
	"context"

	"github.com/gofiber/fiber/v2"

	typWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/types"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/log"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/router"
	pkgWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/whatsapp"
)

func getDeviceContext(c *fiber.Ctx) (deviceID string, jid string) {
	deviceID = c.Locals("device_id").(string)
	jidVal := c.Locals("device_jid")
	if jidVal != nil {
		jid = jidVal.(string)
	}
	return
}

func BuildHistorySyncRequest(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	var req typWhatsApp.RequestBuildHistorySync
	if err := c.BodyParser(&req); err != nil {
		log.DeviceOpCtx(c, "BuildHistorySyncRequest").Warn("Failed to parse body")
		return router.ResponseBadRequest(c, "Failed to parse body request")
	}

	if req.Count <= 0 {
		req.Count = 25
	}

	log.DeviceOpCtx(c, "BuildHistorySyncRequest").WithField("count", req.Count).Info("Building history sync request")

	err := pkgWhatsApp.WhatsAppBuildHistorySyncRequest(ctx, jid, deviceID, req.Count)
	if err != nil {
		log.DeviceOpCtx(c, "BuildHistorySyncRequest").WithError(err).Error("Failed to build history sync")
		return router.ResponseInternalError(c, err.Error())
	}

	log.DeviceOpCtx(c, "BuildHistorySyncRequest").Info("History sync request sent successfully")

	return router.ResponseSuccessWithData(c, "History sync requested", map[string]interface{}{
		"requested": true,
		"count":     req.Count,
		"note":      "History will be delivered via history.sync webhook event",
	})
}
