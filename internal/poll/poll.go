package poll

import (
	"context"

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

// CreatePoll creates a new poll in a chat
func CreatePoll(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	chatJID := c.Params("chat_jid")

	if err := validation.ValidateChatJID(chatJID); err != nil {
		log.MessageOpCtx(c, "CreatePoll", chatJID).Warn("Invalid chat_jid")
		return router.ResponseBadRequest(c, err.Error())
	}

	var req typWhatsApp.RequestSendPoll
	err := c.BodyParser(&req)
	if err != nil {
		log.MessageOpCtx(c, "CreatePoll", chatJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	if req.Question == "" {
		log.MessageOpCtx(c, "CreatePoll", chatJID).Warn("Question is required")
		return router.ResponseBadRequest(c, "question is required")
	}
	if len(req.Options) < 2 {
		log.MessageOpCtx(c, "CreatePoll", chatJID).Warn("At least 2 options are required")
		return router.ResponseBadRequest(c, "at least 2 options are required")
	}
	if len(req.Options) > 12 {
		log.MessageOpCtx(c, "CreatePoll", chatJID).Warn("Maximum 12 options allowed")
		return router.ResponseBadRequest(c, "maximum 12 options allowed")
	}

	log.MessageOpCtx(c, "CreatePoll", chatJID).WithField("question", req.Question).WithField("options_count", len(req.Options)).Info("Creating poll")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	msgID, err := pkgWhatsApp.WhatsAppCreatePoll(ctx, jid, deviceID, chatJID, req.Question, req.Options, req.MultiAnswer)
	if err != nil {
		log.MessageOpCtx(c, "CreatePoll", chatJID).WithError(err).Error("Failed to create poll")
		return router.ResponseInternalError(c, err.Error())
	}

	log.MessageOpCtx(c, "CreatePoll", chatJID).WithField("message_id", msgID).Info("Poll created successfully")

	return router.ResponseSuccessWithData(c, "Success create poll", map[string]interface{}{"message_id": msgID})
}

// VotePoll votes on an existing poll
func VotePoll(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	pollID := c.Params("poll_id")

	var req struct {
		ChatJID string   `json:"chat_jid"`
		Options []string `json:"options"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.Poll(c, "VotePoll", pollID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	if req.ChatJID == "" {
		log.Poll(c, "VotePoll", pollID).Warn("chat_jid is required")
		return router.ResponseBadRequest(c, "chat_jid is required")
	}
	if len(req.Options) == 0 {
		log.Poll(c, "VotePoll", pollID).Warn("At least 1 option is required")
		return router.ResponseBadRequest(c, "at least 1 option is required to vote")
	}

	log.Poll(c, "VotePoll", pollID).WithField("chat_jid", req.ChatJID).WithField("options", req.Options).Info("Voting on poll")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	err = pkgWhatsApp.WhatsAppVotePoll(ctx, jid, deviceID, req.ChatJID, pollID, req.Options)
	if err != nil {
		log.Poll(c, "VotePoll", pollID).WithError(err).Error("Failed to vote on poll")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Poll(c, "VotePoll", pollID).Info("Poll vote submitted successfully")

	return router.ResponseSuccess(c, "Success vote on poll")
}

// GetPollResults gets the results of a poll (placeholder - requires message history)
func GetPollResults(c *fiber.Ctx) error {
	pollID := c.Params("poll_id")

	// Poll results are typically tracked via webhook events when votes come in
	// This endpoint returns the poll message ID for reference
	log.Poll(c, "GetPollResults", pollID).Info("Getting poll results")

	return router.ResponseSuccessWithData(c, "Poll results are delivered via webhook events", map[string]interface{}{
		"poll_id": pollID,
		"note":    "Subscribe to poll.vote webhook events to receive real-time vote updates",
	})
}

// DeletePoll deletes a poll message
func DeletePoll(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	pollID := c.Params("poll_id")

	var req struct {
		ChatJID string `json:"chat_jid"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.Poll(c, "DeletePoll", pollID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	if req.ChatJID == "" {
		log.Poll(c, "DeletePoll", pollID).Warn("chat_jid is required")
		return router.ResponseBadRequest(c, "chat_jid is required")
	}

	log.Poll(c, "DeletePoll", pollID).WithField("chat_jid", req.ChatJID).Info("Deleting poll")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	err = pkgWhatsApp.WhatsAppMessageDelete(ctx, jid, deviceID, req.ChatJID, pollID)
	if err != nil {
		log.Poll(c, "DeletePoll", pollID).WithError(err).Error("Failed to delete poll")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Poll(c, "DeletePoll", pollID).Info("Poll deleted successfully")

	return router.ResponseSuccess(c, "Success delete poll")
}

