package bot

import (
	"context"

	"github.com/gofiber/fiber/v2"

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

// GetBotList retrieves the list of available WhatsApp bots
func GetBotList(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	log.SessionWithDevice(deviceID, jid, "GetBotList").Info("Getting bot list")

	botList, err := pkgWhatsApp.WhatsAppGetBotListV2(ctx, jid, deviceID)
	if err != nil {
		log.SessionWithDevice(deviceID, jid, "GetBotList").
			WithError(err).
			Error("Failed to get bot list")
		return router.ResponseInternalError(c, err.Error())
	}

	// Convert to response format
	bots := make([]map[string]interface{}, 0, len(botList))
	for _, bot := range botList {
		bots = append(bots, map[string]interface{}{
			"jid":        bot.BotJID.String(),
			"persona_id": bot.PersonaID,
		})
	}

	log.SessionWithDevice(deviceID, jid, "GetBotList").
		WithField("count", len(bots)).
		Info("Bot list retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get bot list", map[string]interface{}{
		"bots": bots,
	})
}

// GetBotProfiles retrieves profiles for available bots
func GetBotProfiles(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	log.SessionWithDevice(deviceID, jid, "GetBotProfiles").Info("Getting bot profiles")

	// First get the bot list
	botList, err := pkgWhatsApp.WhatsAppGetBotListV2(ctx, jid, deviceID)
	if err != nil {
		log.SessionWithDevice(deviceID, jid, "GetBotProfiles").
			WithError(err).
			Error("Failed to get bot list")
		return router.ResponseInternalError(c, err.Error())
	}

	if len(botList) == 0 {
		return router.ResponseSuccessWithData(c, "No bots available", map[string]interface{}{
			"profiles": []interface{}{},
		})
	}

	// Get profiles for the bots
	profiles, err := pkgWhatsApp.WhatsAppGetBotProfiles(ctx, jid, deviceID, botList)
	if err != nil {
		log.SessionWithDevice(deviceID, jid, "GetBotProfiles").
			WithError(err).
			Error("Failed to get bot profiles")
		return router.ResponseInternalError(c, err.Error())
	}

	// Convert to response format
	result := make([]map[string]interface{}, 0, len(profiles))
	for _, profile := range profiles {
		result = append(result, map[string]interface{}{
			"jid":         profile.JID.String(),
			"persona_id":  profile.PersonaID,
			"name":        profile.Name,
			"description": profile.Description,
			"commands":    profile.Commands,
		})
	}

	log.SessionWithDevice(deviceID, jid, "GetBotProfiles").
		WithField("count", len(result)).
		Info("Bot profiles retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get bot profiles", map[string]interface{}{
		"profiles": result,
	})
}
