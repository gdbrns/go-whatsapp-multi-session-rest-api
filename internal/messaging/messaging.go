package messaging

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"

	"github.com/gofiber/fiber/v2"

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
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	msgID, err := pkgWhatsApp.WhatsAppSendText(ctx, jid, deviceID, chatJID, reqSendMessage.Text)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Success send message", map[string]interface{}{"message_id": msgID})
}


func SendImage(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	chatJID := c.Params("chat_jid")

	caption := c.FormValue("caption")
	viewOnce := c.FormValue("view_once") == "true"

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return router.ResponseBadRequest(c, "file is required")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}
	defer file.Close()

	fileBytes, err := convertFileToBytes(file)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	msgID, err := pkgWhatsApp.WhatsAppSendImage(ctx, jid, deviceID, chatJID, fileBytes, "image/jpeg", caption, viewOnce)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Success send image", map[string]interface{}{"message_id": msgID})
}

func SendDocument(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	chatJID := c.Params("chat_jid")

	fileName := c.FormValue("filename")

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return router.ResponseBadRequest(c, "file is required")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}
	defer file.Close()

	fileBytes, err := convertFileToBytes(file)
	if err != nil {
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
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Success send document", map[string]interface{}{"message_id": msgID})
}

func GetMessages(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	chatJID := c.Params("chat_jid")

	limit := c.QueryInt("limit", 50)
	before := c.Query("before", "")
	after := c.Query("after", "")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	chatID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, chatJID)

	messages, err := pkgWhatsApp.WhatsAppGetChatHistory(jid, deviceID, chatID, limit, before, after)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

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
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	chatID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, chatJID)

	err = pkgWhatsApp.WhatsAppArchiveChat(ctx, jid, deviceID, chatID, req.Archive)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

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
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	chatID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, chatJID)

	err = pkgWhatsApp.WhatsAppPinChat(ctx, jid, deviceID, chatID, req.Pin)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccess(c, "Success pin chat")
}
