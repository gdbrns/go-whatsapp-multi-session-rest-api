package messaging

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"

	"github.com/gofiber/fiber/v2"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/log"
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

func convertFileToBytes(file multipart.File) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	_, err := io.Copy(buffer, file)
	if err != nil {
		return bytes.NewBuffer(nil).Bytes(), err
	}
	return buffer.Bytes(), nil
}

func SendText(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	chatJID := c.Params("chat_jid")

	var reqSendMessage typWhatsApp.RequestSendMessage
	err := c.BodyParser(&reqSendMessage)
	if err != nil {
		log.MessageOp(deviceID, jid, "SendText", chatJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.MessageOp(deviceID, jid, "SendText", chatJID).WithField("text_length", len(reqSendMessage.Text)).Info("Sending text message")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	msgID, err := pkgWhatsApp.WhatsAppSendText(ctx, jid, deviceID, chatJID, reqSendMessage.Text)
	if err != nil {
		log.MessageOp(deviceID, jid, "SendText", chatJID).WithError(err).Error("Failed to send text message")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOp(deviceID, jid, "SendText", chatJID).WithField("message_id", msgID).Info("Text message sent successfully")

	return router.ResponseSuccessWithData(c, "Success send message", map[string]interface{}{"message_id": msgID})
}


func SendImage(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	chatJID := c.Params("chat_jid")

	caption := c.FormValue("caption")
	viewOnce := c.FormValue("view_once") == "true"

	fileHeader, err := c.FormFile("file")
	if err != nil {
		log.MessageOp(deviceID, jid, "SendImage", chatJID).Warn("No file provided")
		return router.ResponseBadRequest(c, "file is required")
	}

	log.MessageOp(deviceID, jid, "SendImage", chatJID).WithField("filename", fileHeader.Filename).WithField("size", fileHeader.Size).WithField("view_once", viewOnce).Info("Sending image")

	file, err := fileHeader.Open()
	if err != nil {
		log.MessageOp(deviceID, jid, "SendImage", chatJID).WithError(err).Error("Failed to open file")
		return router.ResponseInternalError(c, err.Error())
	}
	defer file.Close()

	fileBytes, err := convertFileToBytes(file)
	if err != nil {
		log.MessageOp(deviceID, jid, "SendImage", chatJID).WithError(err).Error("Failed to convert file to bytes")
		return router.ResponseInternalError(c, err.Error())
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	msgID, err := pkgWhatsApp.WhatsAppSendImage(ctx, jid, deviceID, chatJID, fileBytes, "image/jpeg", caption, viewOnce)
	if err != nil {
		log.MessageOp(deviceID, jid, "SendImage", chatJID).WithError(err).Error("Failed to send image")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOp(deviceID, jid, "SendImage", chatJID).WithField("message_id", msgID).Info("Image sent successfully")

	return router.ResponseSuccessWithData(c, "Success send image", map[string]interface{}{"message_id": msgID})
}

func SendDocument(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	chatJID := c.Params("chat_jid")

	fileName := c.FormValue("filename")

	fileHeader, err := c.FormFile("file")
	if err != nil {
		log.MessageOp(deviceID, jid, "SendDocument", chatJID).Warn("No file provided")
		return router.ResponseBadRequest(c, "file is required")
	}

	log.MessageOp(deviceID, jid, "SendDocument", chatJID).WithField("filename", fileHeader.Filename).WithField("size", fileHeader.Size).Info("Sending document")

	file, err := fileHeader.Open()
	if err != nil {
		log.MessageOp(deviceID, jid, "SendDocument", chatJID).WithError(err).Error("Failed to open file")
		return router.ResponseInternalError(c, err.Error())
	}
	defer file.Close()

	fileBytes, err := convertFileToBytes(file)
	if err != nil {
		log.MessageOp(deviceID, jid, "SendDocument", chatJID).WithError(err).Error("Failed to convert file to bytes")
		return router.ResponseInternalError(c, err.Error())
	}

	if fileName == "" {
		fileName = fileHeader.Filename
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	msgID, err := pkgWhatsApp.WhatsAppSendDocument(ctx, jid, deviceID, chatJID, fileBytes, "application/octet-stream", fileName)
	if err != nil {
		log.MessageOp(deviceID, jid, "SendDocument", chatJID).WithError(err).Error("Failed to send document")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOp(deviceID, jid, "SendDocument", chatJID).WithField("message_id", msgID).WithField("filename", fileName).Info("Document sent successfully")

	return router.ResponseSuccessWithData(c, "Success send document", map[string]interface{}{"message_id": msgID})
}

func GetMessages(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	chatJID := c.Params("chat_jid")

	limit := c.QueryInt("limit", 50)
	before := c.Query("before", "")
	after := c.Query("after", "")

	log.MessageOp(deviceID, jid, "GetMessages", chatJID).WithField("limit", limit).Info("Getting chat messages")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	chatID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, chatJID)

	messages, err := pkgWhatsApp.WhatsAppGetChatHistory(jid, deviceID, chatID, limit, before, after)
	if err != nil {
		log.MessageOp(deviceID, jid, "GetMessages", chatJID).WithError(err).Error("Failed to get chat messages")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOp(deviceID, jid, "GetMessages", chatJID).Info("Chat messages retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get chat messages", messages)
}

func ArchiveChat(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	chatJID := c.Params("chat_jid")

	var req struct {
		Archive bool `json:"archive"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.MessageOp(deviceID, jid, "ArchiveChat", chatJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.MessageOp(deviceID, jid, "ArchiveChat", chatJID).WithField("archive", req.Archive).Info("Archiving/unarchiving chat")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	chatID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, chatJID)

	err = pkgWhatsApp.WhatsAppArchiveChat(ctx, jid, deviceID, chatID, req.Archive)
	if err != nil {
		log.MessageOp(deviceID, jid, "ArchiveChat", chatJID).WithError(err).Error("Failed to archive/unarchive chat")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOp(deviceID, jid, "ArchiveChat", chatJID).WithField("archive", req.Archive).Info("Chat archive status updated successfully")

	return router.ResponseSuccess(c, "Success archive chat")
}

func PinChat(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	chatJID := c.Params("chat_jid")

	var req struct {
		Pin bool `json:"pin"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.MessageOp(deviceID, jid, "PinChat", chatJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.MessageOp(deviceID, jid, "PinChat", chatJID).WithField("pin", req.Pin).Info("Pinning/unpinning chat")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	chatID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, chatJID)

	err = pkgWhatsApp.WhatsAppPinChat(ctx, jid, deviceID, chatID, req.Pin)
	if err != nil {
		log.MessageOp(deviceID, jid, "PinChat", chatJID).WithError(err).Error("Failed to pin/unpin chat")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOp(deviceID, jid, "PinChat", chatJID).WithField("pin", req.Pin).Info("Chat pin status updated successfully")

	return router.ResponseSuccess(c, "Success pin chat")
}
