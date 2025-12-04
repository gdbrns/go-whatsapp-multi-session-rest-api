package webhooks

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/webhook"
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

type createWebhookRequest struct {
	URL    string              `json:"url"`
	Events []webhook.EventType `json:"events"`
}

type updateWebhookRequest struct {
	URL    string              `json:"url"`
	Events []webhook.EventType `json:"events"`
	Active bool                `json:"active"`
}

func ListWebhooks(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	log.WebhookOp(deviceID, jid, "ListWebhooks", 0).Info("Listing webhooks")

	engine := pkgWhatsApp.GetWebhookEngine()
	if engine == nil {
		log.WebhookOp(deviceID, jid, "ListWebhooks", 0).Error("Webhook engine not initialized")
		return router.ResponseInternalError(c, "webhook engine not initialized")
	}

	webhooks, err := engine.Store().GetAllWebhooks(context.Background(), deviceID)
	if err != nil {
		log.WebhookOp(deviceID, jid, "ListWebhooks", 0).WithError(err).Error("Failed to list webhooks")
		return router.ResponseInternalError(c, err.Error())
	}

	webhookCount := 0
	if webhooks != nil {
		webhookCount = len(webhooks)
	}

	log.WebhookOp(deviceID, jid, "ListWebhooks", 0).WithField("webhook_count", webhookCount).Info("Webhooks listed successfully")

	return router.ResponseSuccessWithData(c, "success", map[string]interface{}{"webhooks": webhooks})
}

func GetWebhook(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	webhookID, err := c.ParamsInt("webhook_id")
	if err != nil {
		log.WebhookOp(deviceID, jid, "GetWebhook", 0).Warn("Invalid webhook_id parameter")
		return router.ResponseBadRequest(c, "invalid webhook_id")
	}

	log.WebhookOp(deviceID, jid, "GetWebhook", int64(webhookID)).Info("Getting webhook")

	engine := pkgWhatsApp.GetWebhookEngine()
	if engine == nil {
		log.WebhookOp(deviceID, jid, "GetWebhook", int64(webhookID)).Error("Webhook engine not initialized")
		return router.ResponseInternalError(c, "webhook engine not initialized")
	}

	wh, err := engine.Store().GetWebhook(context.Background(), int64(webhookID), deviceID)
	if err != nil {
		log.WebhookOp(deviceID, jid, "GetWebhook", int64(webhookID)).WithError(err).Error("Failed to get webhook")
		return router.ResponseInternalError(c, err.Error())
	}

	log.WebhookOp(deviceID, jid, "GetWebhook", int64(webhookID)).Info("Webhook retrieved successfully")

	return router.ResponseSuccessWithData(c, "success", map[string]interface{}{"webhook": wh})
}

func CreateWebhook(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	var req createWebhookRequest
	if err := c.BodyParser(&req); err != nil {
		log.WebhookOp(deviceID, jid, "CreateWebhook", 0).Warn("Invalid request body")
		return router.ResponseBadRequest(c, "invalid request body")
	}

	if req.URL == "" {
		log.WebhookOp(deviceID, jid, "CreateWebhook", 0).Warn("URL is required")
		return router.ResponseBadRequest(c, "url is required")
	}

	log.WebhookOp(deviceID, jid, "CreateWebhook", 0).WithField("url", req.URL).WithField("event_count", len(req.Events)).Info("Creating webhook")

	secret := make([]byte, 32)
	_, _ = rand.Read(secret)
	secretStr := hex.EncodeToString(secret)

	engine := pkgWhatsApp.GetWebhookEngine()
	if engine == nil {
		log.WebhookOp(deviceID, jid, "CreateWebhook", 0).Error("Webhook engine not initialized")
		return router.ResponseInternalError(c, "webhook engine not initialized")
	}

	webhookID, err := engine.Store().CreateWebhook(context.Background(), deviceID, req.URL, secretStr, req.Events)
	if err != nil {
		log.WebhookOp(deviceID, jid, "CreateWebhook", 0).WithField("url", req.URL).WithError(err).Error("Failed to create webhook")
		return router.ResponseInternalError(c, err.Error())
	}

	log.WebhookOp(deviceID, jid, "CreateWebhook", webhookID).WithField("url", req.URL).Info("Webhook created successfully")

	return router.ResponseSuccessWithData(c, "webhook created", map[string]interface{}{"webhook_id": webhookID, "secret": secretStr})
}

func UpdateWebhook(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	webhookID, err := c.ParamsInt("webhook_id")
	if err != nil {
		log.WebhookOp(deviceID, jid, "UpdateWebhook", 0).Warn("Invalid webhook_id parameter")
		return router.ResponseBadRequest(c, "invalid webhook_id")
	}

	var req updateWebhookRequest
	if err := c.BodyParser(&req); err != nil {
		log.WebhookOp(deviceID, jid, "UpdateWebhook", int64(webhookID)).Warn("Invalid request body")
		return router.ResponseBadRequest(c, "invalid request body")
	}

	if req.URL == "" {
		log.WebhookOp(deviceID, jid, "UpdateWebhook", int64(webhookID)).Warn("URL is required")
		return router.ResponseBadRequest(c, "url is required")
	}

	log.WebhookOp(deviceID, jid, "UpdateWebhook", int64(webhookID)).WithField("url", req.URL).WithField("active", req.Active).Info("Updating webhook")

	engine := pkgWhatsApp.GetWebhookEngine()
	if engine == nil {
		log.WebhookOp(deviceID, jid, "UpdateWebhook", int64(webhookID)).Error("Webhook engine not initialized")
		return router.ResponseInternalError(c, "webhook engine not initialized")
	}

	wh, err := engine.Store().GetWebhook(context.Background(), int64(webhookID), deviceID)
	if err != nil {
		log.WebhookOp(deviceID, jid, "UpdateWebhook", int64(webhookID)).WithError(err).Error("Failed to get existing webhook")
		return router.ResponseInternalError(c, err.Error())
	}

	if err := engine.Store().UpdateWebhook(context.Background(), int64(webhookID), deviceID, req.URL, wh.Secret, req.Events, req.Active); err != nil {
		log.WebhookOp(deviceID, jid, "UpdateWebhook", int64(webhookID)).WithError(err).Error("Failed to update webhook")
		return router.ResponseInternalError(c, err.Error())
	}

	log.WebhookOp(deviceID, jid, "UpdateWebhook", int64(webhookID)).WithField("url", req.URL).WithField("active", req.Active).Info("Webhook updated successfully")

	return router.ResponseSuccess(c, "webhook updated")
}

func DeleteWebhook(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	webhookID, err := c.ParamsInt("webhook_id")
	if err != nil {
		log.WebhookOp(deviceID, jid, "DeleteWebhook", 0).Warn("Invalid webhook_id parameter")
		return router.ResponseBadRequest(c, "invalid webhook_id")
	}

	log.WebhookOp(deviceID, jid, "DeleteWebhook", int64(webhookID)).Info("Deleting webhook")

	engine := pkgWhatsApp.GetWebhookEngine()
	if engine == nil {
		log.WebhookOp(deviceID, jid, "DeleteWebhook", int64(webhookID)).Error("Webhook engine not initialized")
		return router.ResponseInternalError(c, "webhook engine not initialized")
	}

	if err := engine.Store().DeleteWebhook(context.Background(), int64(webhookID), deviceID); err != nil {
		log.WebhookOp(deviceID, jid, "DeleteWebhook", int64(webhookID)).WithError(err).Error("Failed to delete webhook")
		return router.ResponseInternalError(c, err.Error())
	}

	log.WebhookOp(deviceID, jid, "DeleteWebhook", int64(webhookID)).Info("Webhook deleted successfully")

	return router.ResponseSuccess(c, "webhook deleted")
}

func GetWebhookLogs(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	webhookID, err := c.ParamsInt("webhook_id")
	if err != nil {
		log.WebhookOp(deviceID, jid, "GetWebhookLogs", 0).Warn("Invalid webhook_id parameter")
		return router.ResponseBadRequest(c, "invalid webhook_id")
	}

	log.WebhookOp(deviceID, jid, "GetWebhookLogs", int64(webhookID)).Info("Getting webhook logs")

	engine := pkgWhatsApp.GetWebhookEngine()
	if engine == nil {
		log.WebhookOp(deviceID, jid, "GetWebhookLogs", int64(webhookID)).Error("Webhook engine not initialized")
		return router.ResponseInternalError(c, "webhook engine not initialized")
	}

	_, err = engine.Store().GetWebhook(context.Background(), int64(webhookID), deviceID)
	if err != nil {
		log.WebhookOp(deviceID, jid, "GetWebhookLogs", int64(webhookID)).WithError(err).Error("Failed to get webhook")
		return router.ResponseInternalError(c, err.Error())
	}

	logs, err := engine.Store().GetDeliveryLogs(context.Background(), int64(webhookID), 100)
	if err != nil {
		log.WebhookOp(deviceID, jid, "GetWebhookLogs", int64(webhookID)).WithError(err).Error("Failed to get delivery logs")
		return router.ResponseInternalError(c, err.Error())
	}

	logCount := 0
	if logs != nil {
		logCount = len(logs)
	}

	log.WebhookOp(deviceID, jid, "GetWebhookLogs", int64(webhookID)).WithField("log_count", logCount).Info("Webhook logs retrieved successfully")

	return router.ResponseSuccessWithData(c, "success", map[string]interface{}{"logs": logs})
}

func TestWebhook(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	webhookID, err := c.ParamsInt("webhook_id")
	if err != nil {
		log.WebhookOp(deviceID, jid, "TestWebhook", 0).Warn("Invalid webhook_id parameter")
		return router.ResponseBadRequest(c, "invalid webhook_id")
	}

	log.WebhookOp(deviceID, jid, "TestWebhook", int64(webhookID)).Info("Testing webhook")

	engine := pkgWhatsApp.GetWebhookEngine()
	if engine == nil {
		log.WebhookOp(deviceID, jid, "TestWebhook", int64(webhookID)).Error("Webhook engine not initialized")
		return router.ResponseInternalError(c, "webhook engine not initialized")
	}

	_, errWebhook := engine.Store().GetWebhook(context.Background(), int64(webhookID), deviceID)
	if errWebhook != nil {
		log.WebhookOp(deviceID, jid, "TestWebhook", int64(webhookID)).WithError(errWebhook).Error("Failed to get webhook")
		return router.ResponseInternalError(c, errWebhook.Error())
	}

	testEvent := webhook.WebhookEvent{
		EventType: "test.ping",
		DeviceID:  deviceID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"message": "test webhook delivery",
		},
	}

	engine.Dispatch(context.Background(), deviceID, testEvent)

	log.WebhookOp(deviceID, jid, "TestWebhook", int64(webhookID)).Info("Test webhook dispatched successfully")

	return router.ResponseSuccess(c, "test webhook dispatched")
}
