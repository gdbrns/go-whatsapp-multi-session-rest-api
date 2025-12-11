package status

import (
	"bytes"
	"context"
	"io"

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

func convertFileToBytes(file io.Reader) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	_, err := io.Copy(buffer, file)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// PostStatus posts a new status (story)
func PostStatus(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	log.Status(c, "PostStatus").Info("Posting status")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	// Check content type to determine if it's text, image, or video
	contentType := c.Get("Content-Type")

	var msgID string
	var err error

	if contentType == "application/json" || contentType == "" {
		// Text status
		var req struct {
			Text            string `json:"text"`
			BackgroundColor string `json:"background_color"`
			Font            int    `json:"font"`
		}
		if err := c.BodyParser(&req); err != nil {
			log.Status(c, "PostStatus").Warn("Failed to parse body request")
			return router.ResponseBadRequest(c, "Failed parse body request")
		}

		if req.Text == "" {
			log.Status(c, "PostStatus").Warn("Text is required for text status")
			return router.ResponseBadRequest(c, "text is required")
		}

		msgID, err = pkgWhatsApp.WhatsAppPostTextStatus(ctx, jid, deviceID, req.Text, req.BackgroundColor, req.Font)
	} else {
		// Media status (image or video)
		fileHeader, fileErr := c.FormFile("file")
		if fileErr != nil {
			log.Status(c, "PostStatus").Warn("No file provided for media status")
			return router.ResponseBadRequest(c, "file is required for media status")
		}

		file, openErr := fileHeader.Open()
		if openErr != nil {
			log.Status(c, "PostStatus").WithError(openErr).Error("Failed to open file")
			return router.ResponseInternalError(c, openErr.Error())
		}
		defer file.Close()

		fileBytes, readErr := convertFileToBytes(file)
		if readErr != nil {
			log.Status(c, "PostStatus").WithError(readErr).Error("Failed to read file")
			return router.ResponseInternalError(c, readErr.Error())
		}

		caption := c.FormValue("caption")
		mediaType := c.FormValue("type") // "image" or "video"

		if mediaType == "video" {
			msgID, err = pkgWhatsApp.WhatsAppPostVideoStatus(ctx, jid, deviceID, fileBytes, caption)
		} else {
			msgID, err = pkgWhatsApp.WhatsAppPostImageStatus(ctx, jid, deviceID, fileBytes, caption)
		}
	}

	if err != nil {
		log.Status(c, "PostStatus").WithError(err).Error("Failed to post status")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Status(c, "PostStatus").WithField("message_id", msgID).Info("Status posted successfully")

	return router.ResponseSuccessWithData(c, "Success post status", map[string]interface{}{"message_id": msgID})
}

// GetStatusUpdates gets status updates from contacts
func GetStatusUpdates(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	log.Status(c, "GetStatusUpdates").Info("Getting status updates")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	updates, err := pkgWhatsApp.WhatsAppGetStatusUpdates(ctx, jid, deviceID)
	if err != nil {
		log.Status(c, "GetStatusUpdates").WithError(err).Error("Failed to get status updates")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Status(c, "GetStatusUpdates").WithField("count", len(updates)).Info("Status updates retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get status updates", updates)
}

// DeleteStatus deletes own status
func DeleteStatus(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	statusID := c.Params("status_id")

	log.Status(c, "DeleteStatus").WithField("status_id", statusID).Info("Deleting status")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	err := pkgWhatsApp.WhatsAppDeleteStatus(ctx, jid, deviceID, statusID)
	if err != nil {
		log.Status(c, "DeleteStatus").WithError(err).Error("Failed to delete status")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Status(c, "DeleteStatus").Info("Status deleted successfully")

	return router.ResponseSuccess(c, "Success delete status")
}

// GetUserStatus gets status updates from a specific user
func GetUserStatus(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	userJID := c.Params("user_jid")

	log.Status(c, "GetUserStatus").WithField("user_jid", userJID).Info("Getting user status")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	statuses, err := pkgWhatsApp.WhatsAppGetUserStatus(ctx, jid, deviceID, userJID)
	if err != nil {
		log.Status(c, "GetUserStatus").WithError(err).Error("Failed to get user status")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Status(c, "GetUserStatus").WithField("count", len(statuses)).Info("User status retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get user status", statuses)
}

