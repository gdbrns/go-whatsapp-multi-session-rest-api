package message

import (
	"context"

	"github.com/gofiber/fiber/v2"

	typWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/types"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/router"
	pkgWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/whatsapp"
	"go.mau.fi/whatsmeow/types"
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

func MarkRead(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	messageID := c.Params("message_id")

	var reqRead typWhatsApp.RequestMarkRead
	reqRead.MessageID = messageID

	if err := c.BodyParser(&reqRead); err == nil {
	}

	if reqRead.ChatJID == "" || reqRead.SenderJID == "" {
		return router.ResponseBadRequest(c, "chat_jid and sender_jid are required")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	chatJID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, reqRead.ChatJID)
	senderJID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, reqRead.SenderJID)

	err := pkgWhatsApp.WhatsAppMarkRead(jid, deviceID, chatJID, senderJID, reqRead.MessageID)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccess(c, "Success mark message as read")
}

func React(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	messageID := c.Params("message_id")

	var reqReact typWhatsApp.RequestReact
	err := c.BodyParser(&reqReact)
	if err != nil {
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	reqReact.MessageID = messageID

	if reqReact.ChatJID == "" {
		return router.ResponseBadRequest(c, "chat_jid is required")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	chatJID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, reqReact.ChatJID)
	senderJID := chatJID

	msgID, err := pkgWhatsApp.WhatsAppReact(ctx, jid, deviceID, chatJID, senderJID, reqReact.MessageID, reqReact.Emoji)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Success react to message", map[string]interface{}{"message_id": msgID})
}

func Edit(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	messageID := c.Params("message_id")

	var reqEdit typWhatsApp.RequestEdit
	err := c.BodyParser(&reqEdit)
	if err != nil {
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	reqEdit.MessageID = messageID

	if reqEdit.ChatJID == "" {
		return router.ResponseBadRequest(c, "chat_jid is required")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	chatJID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, reqEdit.ChatJID)

	msgID, err := pkgWhatsApp.WhatsAppEditMessage(ctx, jid, deviceID, chatJID, reqEdit.MessageID, reqEdit.Text)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Success edit message", map[string]interface{}{"message_id": msgID})
}

func Delete(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	messageID := c.Params("message_id")

	var reqDelete typWhatsApp.RequestDelete
	reqDelete.MessageID = messageID

	if err := c.BodyParser(&reqDelete); err == nil {
	}

	if reqDelete.ChatJID == "" {
		return router.ResponseBadRequest(c, "chat_jid is required")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	chatJID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, reqDelete.ChatJID)
	var senderJID types.JID
	if reqDelete.SenderJID != "" {
		senderJID = pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, reqDelete.SenderJID)
	} else {
		senderJID = chatJID
	}

	err := pkgWhatsApp.WhatsAppDeleteMessage(ctx, jid, deviceID, chatJID, senderJID, reqDelete.MessageID)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccess(c, "Success delete message")
}

func Reply(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	messageID := c.Params("message_id")

	var reqReply typWhatsApp.RequestReply
	err := c.BodyParser(&reqReply)
	if err != nil {
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	reqReply.MessageID = messageID

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	msgID, err := pkgWhatsApp.WhatsAppSendText(ctx, jid, deviceID, reqReply.ChatJID, reqReply.Text)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Success reply to message", map[string]interface{}{"message_id": msgID})
}
