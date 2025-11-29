package webhooks

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/webhook"
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
	URL    string               `json:"url"`
	Events []webhook.EventType  `json:"events"`
}

type updateWebhookRequest struct {
	URL    string               `json:"url"`
	Events []webhook.EventType  `json:"events"`
	Active bool                 `json:"active"`
}

func ListWebhooks(c *fiber.Ctx) error {
	deviceID, _ := getDeviceContext(c)
	engine := pkgWhatsApp.GetWebhookEngine()
	if engine == nil {
		return router.ResponseInternalError(c, "webhook engine not initialized")
	}

	webhooks, err := engine.Store().GetAllWebhooks(context.Background(), deviceID)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "success", map[string]interface{}{"webhooks": webhooks})
}

func GetWebhook(c *fiber.Ctx) error {
	deviceID, _ := getDeviceContext(c)
	webhookID, err := c.ParamsInt("webhook_id")
	if err != nil {
		return router.ResponseBadRequest(c, "invalid webhook_id")
	}

	engine := pkgWhatsApp.GetWebhookEngine()
	if engine == nil {
		return router.ResponseInternalError(c, "webhook engine not initialized")
	}

	wh, err := engine.Store().GetWebhook(context.Background(), int64(webhookID), deviceID)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "success", map[string]interface{}{"webhook": wh})
}

func CreateWebhook(c *fiber.Ctx) error {
	deviceID, _ := getDeviceContext(c)
	var req createWebhookRequest
	if err := c.BodyParser(&req); err != nil {
		return router.ResponseBadRequest(c, "invalid request body")
	}

	if req.URL == "" {
		return router.ResponseBadRequest(c, "url is required")
	}

	secret := make([]byte, 32)
	_, _ = rand.Read(secret)
	secretStr := hex.EncodeToString(secret)

	engine := pkgWhatsApp.GetWebhookEngine()
	if engine == nil {
		return router.ResponseInternalError(c, "webhook engine not initialized")
	}

	webhookID, err := engine.Store().CreateWebhook(context.Background(), deviceID, req.URL, secretStr, req.Events)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "webhook created", map[string]interface{}{"webhook_id": webhookID, "secret": secretStr})
}

func UpdateWebhook(c *fiber.Ctx) error {
	deviceID, _ := getDeviceContext(c)
	webhookID, err := c.ParamsInt("webhook_id")
	if err != nil {
		return router.ResponseBadRequest(c, "invalid webhook_id")
	}

	var req updateWebhookRequest
	if err := c.BodyParser(&req); err != nil {
		return router.ResponseBadRequest(c, "invalid request body")
	}

	if req.URL == "" {
		return router.ResponseBadRequest(c, "url is required")
	}

	engine := pkgWhatsApp.GetWebhookEngine()
	if engine == nil {
		return router.ResponseInternalError(c, "webhook engine not initialized")
	}

	wh, err := engine.Store().GetWebhook(context.Background(), int64(webhookID), deviceID)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	if err := engine.Store().UpdateWebhook(context.Background(), int64(webhookID), deviceID, req.URL, wh.Secret, req.Events, req.Active); err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccess(c, "webhook updated")
}

func DeleteWebhook(c *fiber.Ctx) error {
	deviceID, _ := getDeviceContext(c)
	webhookID, err := c.ParamsInt("webhook_id")
	if err != nil {
		return router.ResponseBadRequest(c, "invalid webhook_id")
	}

	engine := pkgWhatsApp.GetWebhookEngine()
	if engine == nil {
		return router.ResponseInternalError(c, "webhook engine not initialized")
	}

	if err := engine.Store().DeleteWebhook(context.Background(), int64(webhookID), deviceID); err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccess(c, "webhook deleted")
}

func GetWebhookLogs(c *fiber.Ctx) error {
	deviceID, _ := getDeviceContext(c)
	webhookID, err := c.ParamsInt("webhook_id")
	if err != nil {
		return router.ResponseBadRequest(c, "invalid webhook_id")
	}

	engine := pkgWhatsApp.GetWebhookEngine()
	if engine == nil {
		return router.ResponseInternalError(c, "webhook engine not initialized")
	}

	_, err = engine.Store().GetWebhook(context.Background(), int64(webhookID), deviceID)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	logs, err := engine.Store().GetDeliveryLogs(context.Background(), int64(webhookID), 100)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "success", map[string]interface{}{"logs": logs})
}

func TestWebhook(c *fiber.Ctx) error {
	deviceID, _ := getDeviceContext(c)
	webhookID, err := c.ParamsInt("webhook_id")
	if err != nil {
		return router.ResponseBadRequest(c, "invalid webhook_id")
	}

	engine := pkgWhatsApp.GetWebhookEngine()
	if engine == nil {
		return router.ResponseInternalError(c, "webhook engine not initialized")
	}

	_, errWebhook := engine.Store().GetWebhook(context.Background(), int64(webhookID), deviceID)
	if errWebhook != nil {
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

	return router.ResponseSuccess(c, "test webhook dispatched")
}
