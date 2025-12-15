package business

import (
	"context"
	"net/url"

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

// decodeJID URL-decodes a JID parameter
func decodeJID(encoded string) string {
	decoded, err := url.PathUnescape(encoded)
	if err != nil {
		return encoded
	}
	return decoded
}

// GetBusinessProfile retrieves the business profile for a given JID
func GetBusinessProfile(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	targetJID := decodeJID(c.Params("jid"))

	log.SessionWithDevice(deviceID, jid, "GetBusinessProfile").
		WithField("target_jid", targetJID).
		Info("Getting business profile")

	profile, err := pkgWhatsApp.WhatsAppGetBusinessProfile(ctx, jid, deviceID, targetJID)
	if err != nil {
		log.SessionWithDevice(deviceID, jid, "GetBusinessProfile").
			WithField("target_jid", targetJID).
			WithError(err).
			Error("Failed to get business profile")
		return router.ResponseInternalError(c, err.Error())
	}

	if profile == nil {
		return router.ResponseNotFound(c, "Business profile not found")
	}

	// Convert to response format
	response := map[string]interface{}{
		"jid":     profile.JID.String(),
		"address": profile.Address,
		"email":   profile.Email,
	}

	// Add categories if available
	if len(profile.Categories) > 0 {
		categories := make([]map[string]string, 0, len(profile.Categories))
		for _, cat := range profile.Categories {
			categories = append(categories, map[string]string{
				"id":   cat.ID,
				"name": cat.Name,
			})
		}
		response["categories"] = categories
	}

	// Add profile options if available
	if len(profile.ProfileOptions) > 0 {
		response["profile_options"] = profile.ProfileOptions
	}

	// Add business hours if available
	if len(profile.BusinessHours) > 0 {
		hours := make([]map[string]interface{}, 0, len(profile.BusinessHours))
		for _, h := range profile.BusinessHours {
			hours = append(hours, map[string]interface{}{
				"day_of_week": h.DayOfWeek,
				"mode":        h.Mode,
				"open_time":   h.OpenTime,
				"close_time":  h.CloseTime,
			})
		}
		response["business_hours"] = hours
		response["business_hours_timezone"] = profile.BusinessHoursTimeZone
	}

	log.SessionWithDevice(deviceID, jid, "GetBusinessProfile").
		WithField("target_jid", targetJID).
		Info("Business profile retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get business profile", response)
}

// ResolveBusinessMessageLink resolves a business message link (wa.me/message/XXX)
func ResolveBusinessMessageLink(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	code := c.Params("code")

	if code == "" {
		var req typWhatsApp.RequestResolveBusinessLink
		if err := c.BodyParser(&req); err == nil && req.Code != "" {
			code = req.Code
		}
	}

	if code == "" {
		log.SessionWithDevice(deviceID, jid, "ResolveBusinessMessageLink").Warn("Missing code parameter")
		return router.ResponseBadRequest(c, "code parameter is required")
	}

	log.SessionWithDevice(deviceID, jid, "ResolveBusinessMessageLink").
		WithField("code", code).
		Info("Resolving business message link")

	target, err := pkgWhatsApp.WhatsAppResolveBusinessMessageLink(ctx, jid, deviceID, code)
	if err != nil {
		log.SessionWithDevice(deviceID, jid, "ResolveBusinessMessageLink").
			WithField("code", code).
			WithError(err).
			Error("Failed to resolve business message link")
		return router.ResponseInternalError(c, err.Error())
	}

	if target == nil {
		return router.ResponseNotFound(c, "Business message link not found")
	}

	response := map[string]interface{}{
		"jid":            target.JID.String(),
		"push_name":      target.PushName,
		"verified_name":  target.VerifiedName,
		"is_signed":      target.IsSigned,
		"verified_level": target.VerifiedLevel,
		"message":        target.Message,
	}

	log.SessionWithDevice(deviceID, jid, "ResolveBusinessMessageLink").
		WithField("code", code).
		WithField("target_jid", target.JID.String()).
		Info("Business message link resolved successfully")

	return router.ResponseSuccessWithData(c, "Success resolve business message link", response)
}
