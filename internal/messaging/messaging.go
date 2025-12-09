package messaging

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/log"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/router"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/validation"
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

func parseOptionalBool(val string) *bool {
	if val == "" {
		return nil
	}
	switch strings.ToLower(val) {
	case "true", "1", "yes", "on":
		b := true
		return &b
	case "false", "0", "no", "off":
		b := false
		return &b
	default:
		return nil
	}
}

func SendText(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	chatJID := c.Params("chat_jid")

	var reqSendMessage typWhatsApp.RequestSendMessage
	err := c.BodyParser(&reqSendMessage)
	if err != nil {
		log.MessageOpCtx(c, "SendText", chatJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	if err := validation.ValidateChatJID(chatJID); err != nil {
		log.MessageOpCtx(c, "SendText", chatJID).Warn("Invalid chat_jid")
		return router.ResponseBadRequest(c, err.Error())
	}
	if strings.TrimSpace(reqSendMessage.Text) == "" {
		log.MessageOpCtx(c, "SendText", chatJID).Warn("Text is required")
		return router.ResponseBadRequest(c, "text is required")
	}

	log.MessageOpCtx(c, "SendText", chatJID).WithField("text_length", len(reqSendMessage.Text)).Info("Sending text message")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	opts := &pkgWhatsApp.SendOptions{
		TypingSimulation:   reqSendMessage.TypingSimulation,
		PresenceSimulation: reqSendMessage.PresenceSimulation,
	}
	msgID, err := pkgWhatsApp.WhatsAppSendText(ctx, jid, deviceID, chatJID, reqSendMessage.Text, opts)
	if err != nil {
		log.MessageOpCtx(c, "SendText", chatJID).WithError(err).Error("Failed to send text message")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOpCtx(c, "SendText", chatJID).WithField("message_id", msgID).Info("Text message sent successfully")

	return router.ResponseSuccessWithData(c, "Success send message", map[string]interface{}{"message_id": msgID})
}


func SendImage(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	chatJID := c.Params("chat_jid")

	if err := validation.ValidateChatJID(chatJID); err != nil {
		log.MessageOpCtx(c, "SendImage", chatJID).Warn("Invalid chat_jid")
		return router.ResponseBadRequest(c, err.Error())
	}

	caption := c.FormValue("caption")
	viewOnce := c.FormValue("view_once") == "true"
	typingSimulation := parseOptionalBool(c.FormValue("typing_simulation"))
	presenceSimulation := parseOptionalBool(c.FormValue("presence_simulation"))

	fileHeader, err := c.FormFile("file")
	if err != nil {
		log.MessageOpCtx(c, "SendImage", chatJID).Warn("No file provided")
		return router.ResponseBadRequest(c, "file is required")
	}

	log.MessageOpCtx(c, "SendImage", chatJID).WithField("filename", fileHeader.Filename).WithField("size", fileHeader.Size).WithField("view_once", viewOnce).Info("Sending image")

	file, err := fileHeader.Open()
	if err != nil {
		log.MessageOpCtx(c, "SendImage", chatJID).WithError(err).Error("Failed to open file")
		return router.ResponseInternalError(c, err.Error())
	}
	defer file.Close()

	fileBytes, err := convertFileToBytes(file)
	if err != nil {
		log.MessageOpCtx(c, "SendImage", chatJID).WithError(err).Error("Failed to convert file to bytes")
		return router.ResponseInternalError(c, err.Error())
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	opts := &pkgWhatsApp.SendOptions{
		TypingSimulation:   typingSimulation,
		PresenceSimulation: presenceSimulation,
	}
	msgID, err := pkgWhatsApp.WhatsAppSendImage(ctx, jid, deviceID, chatJID, fileBytes, "image/jpeg", caption, viewOnce, opts)
	if err != nil {
		log.MessageOpCtx(c, "SendImage", chatJID).WithError(err).Error("Failed to send image")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOpCtx(c, "SendImage", chatJID).WithField("message_id", msgID).Info("Image sent successfully")

	return router.ResponseSuccessWithData(c, "Success send image", map[string]interface{}{"message_id": msgID})
}

func SendDocument(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	chatJID := c.Params("chat_jid")

	if err := validation.ValidateChatJID(chatJID); err != nil {
		log.MessageOpCtx(c, "SendDocument", chatJID).Warn("Invalid chat_jid")
		return router.ResponseBadRequest(c, err.Error())
	}

	fileName := c.FormValue("filename")
	typingSimulation := parseOptionalBool(c.FormValue("typing_simulation"))
	presenceSimulation := parseOptionalBool(c.FormValue("presence_simulation"))

	fileHeader, err := c.FormFile("file")
	if err != nil {
		log.MessageOpCtx(c, "SendDocument", chatJID).Warn("No file provided")
		return router.ResponseBadRequest(c, "file is required")
	}

	log.MessageOpCtx(c, "SendDocument", chatJID).WithField("filename", fileHeader.Filename).WithField("size", fileHeader.Size).Info("Sending document")

	file, err := fileHeader.Open()
	if err != nil {
		log.MessageOpCtx(c, "SendDocument", chatJID).WithError(err).Error("Failed to open file")
		return router.ResponseInternalError(c, err.Error())
	}
	defer file.Close()

	fileBytes, err := convertFileToBytes(file)
	if err != nil {
		log.MessageOpCtx(c, "SendDocument", chatJID).WithError(err).Error("Failed to convert file to bytes")
		return router.ResponseInternalError(c, err.Error())
	}

	if fileName == "" {
		fileName = fileHeader.Filename
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	opts := &pkgWhatsApp.SendOptions{
		TypingSimulation:   typingSimulation,
		PresenceSimulation: presenceSimulation,
	}
	msgID, err := pkgWhatsApp.WhatsAppSendDocument(ctx, jid, deviceID, chatJID, fileBytes, "application/octet-stream", fileName, opts)
	if err != nil {
		log.MessageOpCtx(c, "SendDocument", chatJID).WithError(err).Error("Failed to send document")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOpCtx(c, "SendDocument", chatJID).WithField("message_id", msgID).WithField("filename", fileName).Info("Document sent successfully")

	return router.ResponseSuccessWithData(c, "Success send document", map[string]interface{}{"message_id": msgID})
}

func GetMessages(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	chatJID := c.Params("chat_jid")

	if err := validation.ValidateChatJID(chatJID); err != nil {
		log.MessageOpCtx(c, "GetMessages", chatJID).Warn("Invalid chat_jid")
		return router.ResponseBadRequest(c, err.Error())
	}

	limit := c.QueryInt("limit", 50)
	before := c.Query("before", "")
	after := c.Query("after", "")

	log.MessageOpCtx(c, "GetMessages", chatJID).WithField("limit", limit).Info("Getting chat messages")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	chatID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, chatJID)

	messages, err := pkgWhatsApp.WhatsAppGetChatHistory(jid, deviceID, chatID, limit, before, after)
	if err != nil {
		log.MessageOpCtx(c, "GetMessages", chatJID).WithError(err).Error("Failed to get chat messages")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOpCtx(c, "GetMessages", chatJID).Info("Chat messages retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get chat messages", messages)
}

func ArchiveChat(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	chatJID := c.Params("chat_jid")

	if err := validation.ValidateChatJID(chatJID); err != nil {
		log.MessageOpCtx(c, "ArchiveChat", chatJID).Warn("Invalid chat_jid")
		return router.ResponseBadRequest(c, err.Error())
	}

	var req struct {
		Archive bool `json:"archive"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.MessageOpCtx(c, "ArchiveChat", chatJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.MessageOpCtx(c, "ArchiveChat", chatJID).WithField("archive", req.Archive).Info("Archiving/unarchiving chat")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	chatID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, chatJID)

	err = pkgWhatsApp.WhatsAppArchiveChat(ctx, jid, deviceID, chatID, req.Archive)
	if err != nil {
		log.MessageOpCtx(c, "ArchiveChat", chatJID).WithError(err).Error("Failed to archive/unarchive chat")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOpCtx(c, "ArchiveChat", chatJID).WithField("archive", req.Archive).Info("Chat archive status updated successfully")

	return router.ResponseSuccess(c, "Success archive chat")
}

func PinChat(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	chatJID := c.Params("chat_jid")

	if err := validation.ValidateChatJID(chatJID); err != nil {
		log.MessageOpCtx(c, "PinChat", chatJID).Warn("Invalid chat_jid")
		return router.ResponseBadRequest(c, err.Error())
	}

	var req struct {
		Pin bool `json:"pin"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.MessageOpCtx(c, "PinChat", chatJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.MessageOpCtx(c, "PinChat", chatJID).WithField("pin", req.Pin).Info("Pinning/unpinning chat")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	chatID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, chatJID)

	err = pkgWhatsApp.WhatsAppPinChat(ctx, jid, deviceID, chatID, req.Pin)
	if err != nil {
		log.MessageOpCtx(c, "PinChat", chatJID).WithError(err).Error("Failed to pin/unpin chat")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOpCtx(c, "PinChat", chatJID).WithField("pin", req.Pin).Info("Chat pin status updated successfully")

	return router.ResponseSuccess(c, "Success pin chat")
}
