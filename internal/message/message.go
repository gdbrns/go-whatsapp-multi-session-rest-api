package message

import (
	"context"

	"github.com/gofiber/fiber/v2"

	typWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/types"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/log"
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
		log.MessageOp(deviceID, jid, "MarkRead", "").Warn("Missing chat_jid or sender_jid")
		return router.ResponseBadRequest(c, "chat_jid and sender_jid are required")
	}

	log.MessageOp(deviceID, jid, "MarkRead", reqRead.ChatJID).WithField("message_id", messageID).Info("Marking message as read")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	chatJID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, reqRead.ChatJID)
	senderJID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, reqRead.SenderJID)

	err := pkgWhatsApp.WhatsAppMarkRead(jid, deviceID, chatJID, senderJID, reqRead.MessageID)
	if err != nil {
		log.MessageOp(deviceID, jid, "MarkRead", reqRead.ChatJID).WithField("message_id", messageID).WithError(err).Error("Failed to mark message as read")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOp(deviceID, jid, "MarkRead", reqRead.ChatJID).WithField("message_id", messageID).Info("Message marked as read successfully")

	return router.ResponseSuccess(c, "Success mark message as read")
}

func React(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	messageID := c.Params("message_id")

	var reqReact typWhatsApp.RequestReact
	err := c.BodyParser(&reqReact)
	if err != nil {
		log.MessageOp(deviceID, jid, "React", "").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	reqReact.MessageID = messageID

	if reqReact.ChatJID == "" {
		log.MessageOp(deviceID, jid, "React", "").Warn("Missing chat_jid")
		return router.ResponseBadRequest(c, "chat_jid is required")
	}

	log.MessageOp(deviceID, jid, "React", reqReact.ChatJID).WithField("message_id", messageID).WithField("emoji", reqReact.Emoji).Info("Reacting to message")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	chatJID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, reqReact.ChatJID)
	senderJID := chatJID

	msgID, err := pkgWhatsApp.WhatsAppReact(ctx, jid, deviceID, chatJID, senderJID, reqReact.MessageID, reqReact.Emoji)
	if err != nil {
		log.MessageOp(deviceID, jid, "React", reqReact.ChatJID).WithField("message_id", messageID).WithError(err).Error("Failed to react to message")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOp(deviceID, jid, "React", reqReact.ChatJID).WithField("message_id", messageID).WithField("reaction_msg_id", msgID).Info("Reaction sent successfully")

	return router.ResponseSuccessWithData(c, "Success react to message", map[string]interface{}{"message_id": msgID})
}

func Edit(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	messageID := c.Params("message_id")

	var reqEdit typWhatsApp.RequestEdit
	err := c.BodyParser(&reqEdit)
	if err != nil {
		log.MessageOp(deviceID, jid, "Edit", "").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	reqEdit.MessageID = messageID

	if reqEdit.ChatJID == "" {
		log.MessageOp(deviceID, jid, "Edit", "").Warn("Missing chat_jid")
		return router.ResponseBadRequest(c, "chat_jid is required")
	}

	log.MessageOp(deviceID, jid, "Edit", reqEdit.ChatJID).WithField("message_id", messageID).Info("Editing message")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	chatJID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, reqEdit.ChatJID)

	msgID, err := pkgWhatsApp.WhatsAppEditMessage(ctx, jid, deviceID, chatJID, reqEdit.MessageID, reqEdit.Text)
	if err != nil {
		log.MessageOp(deviceID, jid, "Edit", reqEdit.ChatJID).WithField("message_id", messageID).WithError(err).Error("Failed to edit message")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOp(deviceID, jid, "Edit", reqEdit.ChatJID).WithField("message_id", messageID).WithField("new_msg_id", msgID).Info("Message edited successfully")

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
		log.MessageOp(deviceID, jid, "Delete", "").Warn("Missing chat_jid")
		return router.ResponseBadRequest(c, "chat_jid is required")
	}

	log.MessageOp(deviceID, jid, "Delete", reqDelete.ChatJID).WithField("message_id", messageID).Info("Deleting message")

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
		log.MessageOp(deviceID, jid, "Delete", reqDelete.ChatJID).WithField("message_id", messageID).WithError(err).Error("Failed to delete message")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOp(deviceID, jid, "Delete", reqDelete.ChatJID).WithField("message_id", messageID).Info("Message deleted successfully")

	return router.ResponseSuccess(c, "Success delete message")
}

func Reply(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	messageID := c.Params("message_id")

	var reqReply typWhatsApp.RequestReply
	err := c.BodyParser(&reqReply)
	if err != nil {
		log.MessageOp(deviceID, jid, "Reply", "").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	reqReply.MessageID = messageID

	log.MessageOp(deviceID, jid, "Reply", reqReply.ChatJID).WithField("reply_to_message_id", messageID).Info("Replying to message")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	msgID, err := pkgWhatsApp.WhatsAppSendText(ctx, jid, deviceID, reqReply.ChatJID, reqReply.Text)
	if err != nil {
		log.MessageOp(deviceID, jid, "Reply", reqReply.ChatJID).WithField("reply_to_message_id", messageID).WithError(err).Error("Failed to reply to message")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOp(deviceID, jid, "Reply", reqReply.ChatJID).WithField("reply_to_message_id", messageID).WithField("new_msg_id", msgID).Info("Reply sent successfully")

	return router.ResponseSuccessWithData(c, "Success reply to message", map[string]interface{}{"message_id": msgID})
}
