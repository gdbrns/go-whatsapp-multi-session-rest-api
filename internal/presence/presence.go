package presence

import (
	"context"
	"fmt"

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
		log.SessionWithDevice(deviceID, jid, "SendChatPresence").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.SessionWithDevice(deviceID, jid, "SendChatPresence").WithField("chat_jid", chatJID).WithField("state", reqPresence.State).WithField("media", reqPresence.Media).Info("Sending chat presence")

	phoneJID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, chatJID)

	isComposing := reqPresence.State == "composing"
	isAudio := reqPresence.Media == "audio"

	pkgWhatsApp.WhatsAppComposeStatus(ctx, jid, deviceID, phoneJID, isComposing, isAudio)

	log.SessionWithDevice(deviceID, jid, "SendChatPresence").WithField("chat_jid", chatJID).Info("Chat presence sent successfully")

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
		log.SessionWithDevice(deviceID, jid, "UpdateStatus").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.SessionWithDevice(deviceID, jid, "UpdateStatus").WithField("status", req.Status).Info("Updating presence status")

	isAvailable := req.Status == "available"
	pkgWhatsApp.WhatsAppPresence(ctx, jid, deviceID, isAvailable)

	log.SessionWithDevice(deviceID, jid, "UpdateStatus").WithField("status", req.Status).Info("Presence status updated successfully")

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
		log.SessionWithDevice(deviceID, jid, "SetDisappearingTimer").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.SessionWithDevice(deviceID, jid, "SetDisappearingTimer").WithField("chat_jid", chatJID).WithField("timer", req.Timer).Info("Setting disappearing timer")

	timer, err := parseTimer(req.Timer)
	if err != nil {
		log.SessionWithDevice(deviceID, jid, "SetDisappearingTimer").WithField("timer", req.Timer).Warn("Invalid timer value")
		return router.ResponseBadRequest(c, "Invalid timer value")
	}

	err = pkgWhatsApp.WhatsAppSetDisappearingTimer(ctx, jid, deviceID, timer, chatJID)
	if err != nil {
		log.SessionWithDevice(deviceID, jid, "SetDisappearingTimer").WithField("chat_jid", chatJID).WithError(err).Error("Failed to set disappearing timer")
		return router.ResponseInternalError(c, err.Error())
	}

	log.SessionWithDevice(deviceID, jid, "SetDisappearingTimer").WithField("chat_jid", chatJID).WithField("timer", req.Timer).Info("Disappearing timer set successfully")

	return router.ResponseSuccess(c, "Success set disappearing timer")
}

// SubscribePresence subscribes to presence updates for a specific user
func SubscribePresence(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	var req struct {
		JID string `json:"jid"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.SessionWithDevice(deviceID, jid, "SubscribePresence").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	if req.JID == "" {
		log.SessionWithDevice(deviceID, jid, "SubscribePresence").Warn("Missing jid parameter")
		return router.ResponseBadRequest(c, "jid is required")
	}

	log.SessionWithDevice(deviceID, jid, "SubscribePresence").WithField("target_jid", req.JID).Info("Subscribing to presence updates")

	err = pkgWhatsApp.WhatsAppSubscribePresence(ctx, jid, deviceID, req.JID)
	if err != nil {
		log.SessionWithDevice(deviceID, jid, "SubscribePresence").WithField("target_jid", req.JID).WithError(err).Error("Failed to subscribe to presence")
		return router.ResponseInternalError(c, err.Error())
	}

	log.SessionWithDevice(deviceID, jid, "SubscribePresence").WithField("target_jid", req.JID).Info("Subscribed to presence updates successfully")

	return router.ResponseSuccess(c, "Successfully subscribed to presence updates")
}

// SetPassive sets the client to passive mode
func SetPassive(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	var req struct {
		Passive bool `json:"passive"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.SessionWithDevice(deviceID, jid, "SetPassive").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.SessionWithDevice(deviceID, jid, "SetPassive").WithField("passive", req.Passive).Info("Setting passive mode")

	err = pkgWhatsApp.WhatsAppSetPassive(ctx, jid, deviceID, req.Passive)
	if err != nil {
		log.SessionWithDevice(deviceID, jid, "SetPassive").WithField("passive", req.Passive).WithError(err).Error("Failed to set passive mode")
		return router.ResponseInternalError(c, err.Error())
	}

	log.SessionWithDevice(deviceID, jid, "SetPassive").WithField("passive", req.Passive).Info("Passive mode set successfully")

	return router.ResponseSuccess(c, "Passive mode set successfully")
}
