package message

import (
	"context"
	"encoding/base64"
	"strings"

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

	_ = c.BodyParser(&reqRead)

	if reqRead.ChatJID == "" || reqRead.SenderJID == "" {
		log.MessageOpCtx(c, "MarkRead", "").Warn("Missing chat_jid or sender_jid")
		return router.ResponseBadRequest(c, "chat_jid and sender_jid are required")
	}

	log.MessageOpCtx(c, "MarkRead", reqRead.ChatJID).WithField("message_id", messageID).Info("Marking message as read")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	chatJID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, reqRead.ChatJID)
	senderJID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, reqRead.SenderJID)

	err := pkgWhatsApp.WhatsAppMarkRead(jid, deviceID, chatJID, senderJID, reqRead.MessageID)
	if err != nil {
		log.MessageOpCtx(c, "MarkRead", reqRead.ChatJID).WithField("message_id", messageID).WithError(err).Error("Failed to mark message as read")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOpCtx(c, "MarkRead", reqRead.ChatJID).WithField("message_id", messageID).Info("Message marked as read successfully")

	return router.ResponseSuccess(c, "Success mark message as read")
}

func React(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	messageID := c.Params("message_id")

	var reqReact typWhatsApp.RequestReact
	err := c.BodyParser(&reqReact)
	if err != nil {
		log.MessageOpCtx(c, "React", "").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	reqReact.MessageID = messageID

	if reqReact.ChatJID == "" {
		log.MessageOpCtx(c, "React", "").Warn("Missing chat_jid")
		return router.ResponseBadRequest(c, "chat_jid is required")
	}

	log.MessageOpCtx(c, "React", reqReact.ChatJID).WithField("message_id", messageID).WithField("emoji", reqReact.Emoji).Info("Reacting to message")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	chatJID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, reqReact.ChatJID)
	senderJID := chatJID

	msgID, err := pkgWhatsApp.WhatsAppReact(ctx, jid, deviceID, chatJID, senderJID, reqReact.MessageID, reqReact.Emoji)
	if err != nil {
		log.MessageOpCtx(c, "React", reqReact.ChatJID).WithField("message_id", messageID).WithError(err).Error("Failed to react to message")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOpCtx(c, "React", reqReact.ChatJID).WithField("message_id", messageID).WithField("reaction_msg_id", msgID).Info("Reaction sent successfully")

	return router.ResponseSuccessWithData(c, "Success react to message", map[string]interface{}{"message_id": msgID})
}

func Edit(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	messageID := c.Params("message_id")

	var reqEdit typWhatsApp.RequestEdit
	err := c.BodyParser(&reqEdit)
	if err != nil {
		log.MessageOpCtx(c, "Edit", "").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	reqEdit.MessageID = messageID

	if reqEdit.ChatJID == "" {
		log.MessageOpCtx(c, "Edit", "").Warn("Missing chat_jid")
		return router.ResponseBadRequest(c, "chat_jid is required")
	}

	log.MessageOpCtx(c, "Edit", reqEdit.ChatJID).WithField("message_id", messageID).Info("Editing message")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	chatJID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, reqEdit.ChatJID)

	msgID, err := pkgWhatsApp.WhatsAppEditMessage(ctx, jid, deviceID, chatJID, reqEdit.MessageID, reqEdit.Text)
	if err != nil {
		log.MessageOpCtx(c, "Edit", reqEdit.ChatJID).WithField("message_id", messageID).WithError(err).Error("Failed to edit message")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOpCtx(c, "Edit", reqEdit.ChatJID).WithField("message_id", messageID).WithField("new_msg_id", msgID).Info("Message edited successfully")

	return router.ResponseSuccessWithData(c, "Success edit message", map[string]interface{}{"message_id": msgID})
}

func Delete(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	messageID := c.Params("message_id")

	var reqDelete typWhatsApp.RequestDelete
	reqDelete.MessageID = messageID

	_ = c.BodyParser(&reqDelete)

	if reqDelete.ChatJID == "" {
		log.MessageOpCtx(c, "Delete", "").Warn("Missing chat_jid")
		return router.ResponseBadRequest(c, "chat_jid is required")
	}

	log.MessageOpCtx(c, "Delete", reqDelete.ChatJID).WithField("message_id", messageID).Info("Deleting message")

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
		log.MessageOpCtx(c, "Delete", reqDelete.ChatJID).WithField("message_id", messageID).WithError(err).Error("Failed to delete message")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOpCtx(c, "Delete", reqDelete.ChatJID).WithField("message_id", messageID).Info("Message deleted successfully")

	return router.ResponseSuccess(c, "Success delete message")
}

func Reply(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	messageID := c.Params("message_id")

	var reqReply typWhatsApp.RequestReply
	err := c.BodyParser(&reqReply)
	if err != nil {
		log.MessageOpCtx(c, "Reply", "").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	reqReply.MessageID = messageID

	log.MessageOpCtx(c, "Reply", reqReply.ChatJID).WithField("reply_to_message_id", messageID).Info("Replying to message")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	opts := &pkgWhatsApp.SendOptions{
		TypingSimulation:   reqReply.TypingSimulation,
		PresenceSimulation: reqReply.PresenceSimulation,
	}
	msgID, err := pkgWhatsApp.WhatsAppSendText(ctx, jid, deviceID, reqReply.ChatJID, reqReply.Text, opts)
	if err != nil {
		log.MessageOpCtx(c, "Reply", reqReply.ChatJID).WithField("reply_to_message_id", messageID).WithError(err).Error("Failed to reply to message")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOpCtx(c, "Reply", reqReply.ChatJID).WithField("reply_to_message_id", messageID).WithField("new_msg_id", msgID).Info("Reply sent successfully")

	return router.ResponseSuccessWithData(c, "Success reply to message", map[string]interface{}{"message_id": msgID})
}

// Forward forwards a message to another chat
func Forward(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	messageID := c.Params("message_id")

	var reqForward typWhatsApp.RequestForward
	err := c.BodyParser(&reqForward)
	if err != nil {
		log.MessageOpCtx(c, "Forward", "").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	reqForward.MessageID = messageID

	if reqForward.ToChatJID == "" {
		log.MessageOpCtx(c, "Forward", "").Warn("Missing to_chat_jid")
		return router.ResponseBadRequest(c, "to_chat_jid is required")
	}

	log.MessageOpCtx(c, "Forward", reqForward.ToChatJID).WithField("message_id", messageID).Info("Forwarding message")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	toChatJID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, reqForward.ToChatJID)

	newMsgID, err := pkgWhatsApp.WhatsAppForwardMessage(jid, deviceID, messageID, toChatJID)
	if err != nil {
		log.MessageOpCtx(c, "Forward", reqForward.ToChatJID).WithField("message_id", messageID).WithError(err).Error("Failed to forward message")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOpCtx(c, "Forward", reqForward.ToChatJID).WithField("message_id", messageID).WithField("new_msg_id", newMsgID).Info("Message forwarded successfully")

	return router.ResponseSuccessWithData(c, "Success forward message", map[string]interface{}{"message_id": newMsgID})
}

// SendMediaRetryReceipt sends a media retry receipt for failed media downloads
func SendMediaRetryReceipt(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	var req typWhatsApp.RequestSendMediaRetryReceipt
	if err := c.BodyParser(&req); err != nil {
		log.MessageOpCtx(c, "SendMediaRetryReceipt", "").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed to parse body request")
	}

	// Validation
	if strings.TrimSpace(req.ChatJID) == "" || strings.TrimSpace(req.SenderJID) == "" || strings.TrimSpace(req.MessageID) == "" || strings.TrimSpace(req.MediaKey) == "" {
		log.MessageOpCtx(c, "SendMediaRetryReceipt", req.ChatJID).Warn("Missing required fields")
		return router.ResponseBadRequest(c, "chat_jid, sender_jid, message_id and media_key are required")
	}

	mediaKey, err := base64.StdEncoding.DecodeString(strings.TrimSpace(req.MediaKey))
	if err != nil {
		log.MessageOpCtx(c, "SendMediaRetryReceipt", req.ChatJID).Warn("Invalid media_key (base64)")
		return router.ResponseBadRequest(c, "media_key must be base64")
	}

	log.MessageOpCtx(c, "SendMediaRetryReceipt", req.ChatJID).WithField("message_id", req.MessageID).Info("Sending media retry receipt")

	err = pkgWhatsApp.WhatsAppSendMediaRetryReceipt(ctx, jid, deviceID, req.ChatJID, req.SenderJID, req.MessageID, mediaKey)
	if err != nil {
		log.MessageOpCtx(c, "SendMediaRetryReceipt", req.ChatJID).WithField("message_id", req.MessageID).WithError(err).Error("Failed to send media retry receipt")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOpCtx(c, "SendMediaRetryReceipt", req.ChatJID).WithField("message_id", req.MessageID).Info("Media retry receipt sent successfully")

	return router.ResponseSuccess(c, "Media retry receipt sent")
}
