package call

import (
	"context"

	"github.com/gofiber/fiber/v2"

	typWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/types"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/log"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/router"
	pkgWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/whatsapp"
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

// RejectCall rejects an incoming WhatsApp call
func RejectCall(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	var req typWhatsApp.RequestRejectCall
	if err := c.BodyParser(&req); err != nil {
		log.SessionWithDevice(deviceID, jid, "RejectCall").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	if req.CallFrom == "" || req.CallID == "" {
		log.SessionWithDevice(deviceID, jid, "RejectCall").Warn("Missing required fields")
		return router.ResponseBadRequest(c, "call_from and call_id are required")
	}

	log.SessionWithDevice(deviceID, jid, "RejectCall").
		WithField("call_from", req.CallFrom).
		WithField("call_id", req.CallID).
		Info("Rejecting call")

	err := pkgWhatsApp.WhatsAppRejectCall(ctx, jid, deviceID, req.CallFrom, req.CallID)
	if err != nil {
		log.SessionWithDevice(deviceID, jid, "RejectCall").WithError(err).Error("Failed to reject call")
		return router.ResponseInternalError(c, err.Error())
	}

	log.SessionWithDevice(deviceID, jid, "RejectCall").
		WithField("call_from", req.CallFrom).
		WithField("call_id", req.CallID).
		Info("Call rejected successfully")

	return router.ResponseSuccess(c, "Call rejected successfully")
}
