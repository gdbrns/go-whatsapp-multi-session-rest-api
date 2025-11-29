package appstate

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mau.fi/whatsmeow/appstate"

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

func FetchAppState(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	appStateName := c.Params("name")

	// Validate app state name
	if appStateName == "" {
		return router.ResponseBadRequest(c, "app state name is required")
	}

	// Parse query parameters
	fullSync := c.QueryBool("full_sync", false)
	onlyIfNotSynced := c.QueryBool("only_if_not_synced", false)

	err := pkgWhatsApp.WhatsAppFetchAppState(jid, deviceID, appStateName, fullSync, onlyIfNotSynced)
	if err != nil {
		return router.ResponseInternalError(c, "Failed to fetch app state: "+err.Error())
	}

	return router.ResponseSuccessWithData(c, "Successfully fetched app state", map[string]interface{}{
		"name":             appStateName,
		"full_sync":        fullSync,
		"only_if_not_synced": onlyIfNotSynced,
	})
}

func SendAppState(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	var req struct {
		Name   string                 `json:"name"`
		Action string                 `json:"action"`
		Data   map[string]interface{} `json:"data"`
	}

	err := c.BodyParser(&req)
	if err != nil {
		return router.ResponseBadRequest(c, "Failed to parse request body")
	}

	// Validate required fields
	if req.Name == "" {
		return router.ResponseBadRequest(c, "app state name is required")
	}
	if req.Action == "" {
		return router.ResponseBadRequest(c, "action is required")
	}

	// Build patch info based on action
	var patchInfo appstate.PatchInfo

	switch req.Action {
	case "mute":
		// Handle chat mute/unmute
		chatJID, ok := req.Data["chat_jid"].(string)
		if !ok {
			return router.ResponseBadRequest(c, "chat_jid is required for mute action")
		}
		muted, _ := req.Data["muted"].(bool)
		duration, _ := req.Data["duration"].(float64)

		parsedJID, err := pkgWhatsApp.WhatsAppCheckJID(context.Background(), jid, deviceID, chatJID)
		if err != nil {
			return router.ResponseInternalError(c, "Invalid chat JID: "+err.Error())
		}

		patchInfo = appstate.BuildMute(parsedJID, muted, time.Duration(duration)*time.Second)

	case "pin":
		// Handle chat pin/unpin
		chatJID, ok := req.Data["chat_jid"].(string)
		if !ok {
			return router.ResponseBadRequest(c, "chat_jid is required for pin action")
		}
		pinned, _ := req.Data["pinned"].(bool)

		parsedJID, err := pkgWhatsApp.WhatsAppCheckJID(context.Background(), jid, deviceID, chatJID)
		if err != nil {
			return router.ResponseInternalError(c, "Invalid chat JID: "+err.Error())
		}

		patchInfo = appstate.BuildPin(parsedJID, pinned)

	default:
		return router.ResponseBadRequest(c, "unsupported action: "+req.Action)
	}

	err = pkgWhatsApp.WhatsAppSendAppState(jid, deviceID, patchInfo)
	if err != nil {
		return router.ResponseInternalError(c, "Failed to send app state: "+err.Error())
	}

	return router.ResponseSuccessWithData(c, "Successfully sent app state", map[string]interface{}{
		"name":   req.Name,
		"action": req.Action,
	})
}

func MarkNotDirty(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	var req struct {
		CleanType string `json:"clean_type"`
		Timestamp int64  `json:"timestamp"`
	}

	err := c.BodyParser(&req)
	if err != nil {
		return router.ResponseBadRequest(c, "Failed to parse request body")
	}

	// Validate required fields
	if req.CleanType == "" {
		return router.ResponseBadRequest(c, "clean_type is required")
	}

	// Use provided timestamp or current time
	var timestamp time.Time
	if req.Timestamp > 0 {
		timestamp = time.Unix(req.Timestamp, 0)
	} else {
		timestamp = time.Now()
	}

	err = pkgWhatsApp.WhatsAppMarkNotDirty(jid, deviceID, req.CleanType, timestamp)
	if err != nil {
		return router.ResponseInternalError(c, "Failed to mark app state as clean: "+err.Error())
	}

	return router.ResponseSuccessWithData(c, "Successfully marked app state as clean", map[string]interface{}{
		"clean_type": req.CleanType,
		"timestamp":  timestamp.Unix(),
	})
}
