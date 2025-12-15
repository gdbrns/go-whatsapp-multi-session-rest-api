package newsletter

import (
	"context"

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

// ListNewsletters lists all subscribed newsletters/channels
func ListNewsletters(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	log.Newsletter(c, "ListNewsletters", "").Info("Listing subscribed newsletters")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	newsletters, err := pkgWhatsApp.WhatsAppGetSubscribedNewsletters(ctx, jid, deviceID)
	if err != nil {
		log.Newsletter(c, "ListNewsletters", "").WithError(err).Error("Failed to list newsletters")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Newsletter(c, "ListNewsletters", "").WithField("count", len(newsletters)).Info("Newsletters listed successfully")

	return router.ResponseSuccessWithData(c, "Success get subscribed newsletters", newsletters)
}

// CreateNewsletter creates a new newsletter/channel
func CreateNewsletter(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.Newsletter(c, "CreateNewsletter", "").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	if req.Name == "" {
		log.Newsletter(c, "CreateNewsletter", "").Warn("Name is required")
		return router.ResponseBadRequest(c, "name is required")
	}

	log.Newsletter(c, "CreateNewsletter", "").WithField("name", req.Name).Info("Creating newsletter")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	newsletter, err := pkgWhatsApp.WhatsAppCreateNewsletter(ctx, jid, deviceID, req.Name, req.Description)
	if err != nil {
		log.Newsletter(c, "CreateNewsletter", "").WithError(err).Error("Failed to create newsletter")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Newsletter(c, "CreateNewsletter", "").Info("Newsletter created successfully")

	return router.ResponseSuccessWithData(c, "Success create newsletter", newsletter)
}

// GetNewsletterInfo gets information about a specific newsletter
func GetNewsletterInfo(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	newsletterJID := c.Params("jid")

	log.Newsletter(c, "GetNewsletterInfo", newsletterJID).Info("Getting newsletter info")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	info, err := pkgWhatsApp.WhatsAppGetNewsletterInfo(ctx, jid, deviceID, newsletterJID)
	if err != nil {
		log.Newsletter(c, "GetNewsletterInfo", newsletterJID).WithError(err).Error("Failed to get newsletter info")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Newsletter(c, "GetNewsletterInfo", newsletterJID).Info("Newsletter info retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get newsletter info", info)
}

// FollowNewsletter subscribes to a newsletter
func FollowNewsletter(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	newsletterJID := c.Params("jid")

	log.Newsletter(c, "FollowNewsletter", newsletterJID).Info("Following newsletter")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	err := pkgWhatsApp.WhatsAppFollowNewsletter(ctx, jid, deviceID, newsletterJID)
	if err != nil {
		log.Newsletter(c, "FollowNewsletter", newsletterJID).WithError(err).Error("Failed to follow newsletter")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Newsletter(c, "FollowNewsletter", newsletterJID).Info("Newsletter followed successfully")

	return router.ResponseSuccess(c, "Success follow newsletter")
}

// UnfollowNewsletter unsubscribes from a newsletter
func UnfollowNewsletter(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	newsletterJID := c.Params("jid")

	log.Newsletter(c, "UnfollowNewsletter", newsletterJID).Info("Unfollowing newsletter")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	err := pkgWhatsApp.WhatsAppUnfollowNewsletter(ctx, jid, deviceID, newsletterJID)
	if err != nil {
		log.Newsletter(c, "UnfollowNewsletter", newsletterJID).WithError(err).Error("Failed to unfollow newsletter")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Newsletter(c, "UnfollowNewsletter", newsletterJID).Info("Newsletter unfollowed successfully")

	return router.ResponseSuccess(c, "Success unfollow newsletter")
}

// GetNewsletterMessages gets messages from a newsletter
func GetNewsletterMessages(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	newsletterJID := c.Params("jid")

	count := c.QueryInt("count", 50)
	before := c.QueryInt("before", 0)

	log.Newsletter(c, "GetNewsletterMessages", newsletterJID).WithField("count", count).Info("Getting newsletter messages")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	messages, err := pkgWhatsApp.WhatsAppGetNewsletterMessages(ctx, jid, deviceID, newsletterJID, count, before)
	if err != nil {
		log.Newsletter(c, "GetNewsletterMessages", newsletterJID).WithError(err).Error("Failed to get newsletter messages")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Newsletter(c, "GetNewsletterMessages", newsletterJID).WithField("count", len(messages)).Info("Newsletter messages retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get newsletter messages", messages)
}

// SendNewsletterMessage sends a message to a newsletter (admin only)
func SendNewsletterMessage(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	newsletterJID := c.Params("jid")

	var req struct {
		Text string `json:"text"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.Newsletter(c, "SendNewsletterMessage", newsletterJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	if req.Text == "" {
		log.Newsletter(c, "SendNewsletterMessage", newsletterJID).Warn("Text is required")
		return router.ResponseBadRequest(c, "text is required")
	}

	log.Newsletter(c, "SendNewsletterMessage", newsletterJID).WithField("text_length", len(req.Text)).Info("Sending newsletter message")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	msgID, err := pkgWhatsApp.WhatsAppSendNewsletterMessage(ctx, jid, deviceID, newsletterJID, req.Text)
	if err != nil {
		log.Newsletter(c, "SendNewsletterMessage", newsletterJID).WithError(err).Error("Failed to send newsletter message")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Newsletter(c, "SendNewsletterMessage", newsletterJID).WithField("message_id", msgID).Info("Newsletter message sent successfully")

	return router.ResponseSuccessWithData(c, "Success send newsletter message", map[string]interface{}{"message_id": msgID})
}

// ReactToNewsletterMessage reacts to a newsletter message
func ReactToNewsletterMessage(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	newsletterJID := c.Params("jid")

	var req struct {
		MessageServerID int    `json:"message_server_id"`
		Emoji           string `json:"emoji"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.Newsletter(c, "ReactToNewsletterMessage", newsletterJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	if req.MessageServerID == 0 {
		log.Newsletter(c, "ReactToNewsletterMessage", newsletterJID).Warn("message_server_id is required")
		return router.ResponseBadRequest(c, "message_server_id is required")
	}

	log.Newsletter(c, "ReactToNewsletterMessage", newsletterJID).WithField("message_server_id", req.MessageServerID).WithField("emoji", req.Emoji).Info("Reacting to newsletter message")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	err = pkgWhatsApp.WhatsAppNewsletterSendReaction(ctx, jid, deviceID, newsletterJID, req.MessageServerID, req.Emoji)
	if err != nil {
		log.Newsletter(c, "ReactToNewsletterMessage", newsletterJID).WithError(err).Error("Failed to react to newsletter message")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Newsletter(c, "ReactToNewsletterMessage", newsletterJID).Info("Newsletter reaction sent successfully")

	return router.ResponseSuccess(c, "Success react to newsletter message")
}

// ToggleNewsletterMute mutes or unmutes a newsletter
func ToggleNewsletterMute(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	newsletterJID := c.Params("jid")

	var req struct {
		Mute bool `json:"mute"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.Newsletter(c, "ToggleNewsletterMute", newsletterJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.Newsletter(c, "ToggleNewsletterMute", newsletterJID).WithField("mute", req.Mute).Info("Toggling newsletter mute")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	err = pkgWhatsApp.WhatsAppNewsletterToggleMute(ctx, jid, deviceID, newsletterJID, req.Mute)
	if err != nil {
		log.Newsletter(c, "ToggleNewsletterMute", newsletterJID).WithError(err).Error("Failed to toggle newsletter mute")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Newsletter(c, "ToggleNewsletterMute", newsletterJID).Info("Newsletter mute toggled successfully")

	return router.ResponseSuccess(c, "Success toggle newsletter mute")
}

// MarkNewsletterViewed marks newsletter messages as viewed
func MarkNewsletterViewed(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	newsletterJID := c.Params("jid")

	var req struct {
		MessageServerIDs []int `json:"message_server_ids"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.Newsletter(c, "MarkNewsletterViewed", newsletterJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	if len(req.MessageServerIDs) == 0 {
		log.Newsletter(c, "MarkNewsletterViewed", newsletterJID).Warn("message_server_ids is required")
		return router.ResponseBadRequest(c, "message_server_ids is required")
	}

	log.Newsletter(c, "MarkNewsletterViewed", newsletterJID).WithField("count", len(req.MessageServerIDs)).Info("Marking newsletter messages as viewed")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	err = pkgWhatsApp.WhatsAppNewsletterMarkViewed(ctx, jid, deviceID, newsletterJID, req.MessageServerIDs)
	if err != nil {
		log.Newsletter(c, "MarkNewsletterViewed", newsletterJID).WithError(err).Error("Failed to mark newsletter messages as viewed")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Newsletter(c, "MarkNewsletterViewed", newsletterJID).Info("Newsletter messages marked as viewed successfully")

	return router.ResponseSuccess(c, "Success mark newsletter messages as viewed")
}

// GetNewsletterInfoFromInvite gets newsletter info from an invite code
func GetNewsletterInfoFromInvite(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	inviteCode := c.Params("code")

	log.Newsletter(c, "GetNewsletterInfoFromInvite", inviteCode).Info("Getting newsletter info from invite")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	info, err := pkgWhatsApp.WhatsAppGetNewsletterInfoWithInvite(ctx, jid, deviceID, inviteCode)
	if err != nil {
		log.Newsletter(c, "GetNewsletterInfoFromInvite", inviteCode).WithError(err).Error("Failed to get newsletter info from invite")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Newsletter(c, "GetNewsletterInfoFromInvite", inviteCode).Info("Newsletter info retrieved from invite successfully")

	return router.ResponseSuccessWithData(c, "Success get newsletter info from invite", info)
}

// SubscribeLiveUpdates subscribes to live updates for a newsletter
func SubscribeLiveUpdates(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	newsletterJID := c.Params("jid")

	log.Newsletter(c, "SubscribeLiveUpdates", newsletterJID).Info("Subscribing to newsletter live updates")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	err := pkgWhatsApp.WhatsAppNewsletterSubscribeLiveUpdates(ctx, jid, deviceID, newsletterJID)
	if err != nil {
		log.Newsletter(c, "SubscribeLiveUpdates", newsletterJID).WithError(err).Error("Failed to subscribe to newsletter live updates")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Newsletter(c, "SubscribeLiveUpdates", newsletterJID).Info("Subscribed to newsletter live updates successfully")

	return router.ResponseSuccess(c, "Success subscribe to newsletter live updates")
}

// UpdateNewsletterPhoto updates the newsletter photo (admin only)
func UpdateNewsletterPhoto(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	newsletterJID := c.Params("jid")

	fileHeader, err := c.FormFile("file")
	if err != nil {
		log.Newsletter(c, "UpdateNewsletterPhoto", newsletterJID).Warn("No file provided")
		return router.ResponseBadRequest(c, "file is required")
	}

	log.Newsletter(c, "UpdateNewsletterPhoto", newsletterJID).WithField("filename", fileHeader.Filename).Info("Updating newsletter photo")

	file, err := fileHeader.Open()
	if err != nil {
		log.Newsletter(c, "UpdateNewsletterPhoto", newsletterJID).WithError(err).Error("Failed to open file")
		return router.ResponseInternalError(c, err.Error())
	}
	defer file.Close()

	// Read file bytes
	fileBytes := make([]byte, fileHeader.Size)
	_, err = file.Read(fileBytes)
	if err != nil {
		log.Newsletter(c, "UpdateNewsletterPhoto", newsletterJID).WithError(err).Error("Failed to read file")
		return router.ResponseInternalError(c, err.Error())
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	err = pkgWhatsApp.WhatsAppUploadNewsletterPhoto(ctx, jid, deviceID, newsletterJID, fileBytes)
	if err != nil {
		log.Newsletter(c, "UpdateNewsletterPhoto", newsletterJID).WithError(err).Error("Failed to update newsletter photo")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Newsletter(c, "UpdateNewsletterPhoto", newsletterJID).Info("Newsletter photo updated successfully")

	return router.ResponseSuccess(c, "Success update newsletter photo")
}

// GetNewsletterMessageUpdates gets updates for messages in a newsletter (edits, reactions count, etc.)
func GetNewsletterMessageUpdates(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	newsletterJID := c.Params("jid")

	count := c.QueryInt("count", 100)
	since := c.QueryInt("since", 0)

	log.Newsletter(c, "GetNewsletterMessageUpdates", newsletterJID).
		WithField("count", count).
		WithField("since", since).
		Info("Getting newsletter message updates")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	updates, err := pkgWhatsApp.WhatsAppGetNewsletterMessageUpdates(ctx, jid, deviceID, newsletterJID, count, since)
	if err != nil {
		log.Newsletter(c, "GetNewsletterMessageUpdates", newsletterJID).WithError(err).Error("Failed to get newsletter message updates")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Newsletter(c, "GetNewsletterMessageUpdates", newsletterJID).Info("Newsletter message updates retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get newsletter message updates", updates)
}

// AcceptTOSNotice accepts the Terms of Service notice for newsletter features
func AcceptTOSNotice(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	var req struct {
		NoticeID string `json:"notice_id"`
		Stage    string `json:"stage"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.Newsletter(c, "AcceptTOSNotice", "").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	if req.NoticeID == "" {
		log.Newsletter(c, "AcceptTOSNotice", "").Warn("notice_id is required")
		return router.ResponseBadRequest(c, "notice_id is required")
	}

	if req.Stage == "" {
		log.Newsletter(c, "AcceptTOSNotice", "").Warn("stage is required")
		return router.ResponseBadRequest(c, "stage is required")
	}

	log.Newsletter(c, "AcceptTOSNotice", "").
		WithField("notice_id", req.NoticeID).
		WithField("stage", req.Stage).
		Info("Accepting TOS notice")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	err = pkgWhatsApp.WhatsAppAcceptTOSNotice(ctx, jid, deviceID, req.NoticeID, req.Stage)
	if err != nil {
		log.Newsletter(c, "AcceptTOSNotice", "").WithError(err).Error("Failed to accept TOS notice")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Newsletter(c, "AcceptTOSNotice", "").Info("TOS notice accepted successfully")

	return router.ResponseSuccess(c, "Success accept TOS notice")
}

