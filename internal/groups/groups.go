package groups

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	typWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/types"
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

func List(c *fiber.Ctx) error {
	startTotal := time.Now()
	
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	
	resolvePhoneNumbers := c.QueryBool("resolve_phone", false)
	forceRefresh := c.QueryBool("force_refresh", false)

	log.GroupOp(deviceID, jid, "ListGroups", "").WithField("resolve_phone", resolvePhoneNumbers).WithField("force_refresh", forceRefresh).Info("Listing groups")

	startFetch := time.Now()
	groups, err := pkgWhatsApp.WhatsAppGroupList(ctx, jid, deviceID, resolvePhoneNumbers, forceRefresh)
	fetchDuration := time.Since(startFetch)

	if err != nil {
		log.GroupOp(deviceID, jid, "ListGroups", "").WithError(err).Error("Failed to list groups")
		return router.ResponseInternalError(c, err.Error())
	}

	groupCount := 0
	if groups != nil {
		groupCount = len(groups)
	}

	log.GroupOp(deviceID, jid, "ListGroups", "").WithField("group_count", groupCount).WithField("fetch_duration_ms", fetchDuration.Milliseconds()).WithField("total_duration_ms", time.Since(startTotal).Milliseconds()).Info("Groups listed successfully")
	
	return router.ResponseSuccessWithData(c, fmt.Sprintf("Success get %d groups with members", groupCount), groups)
}

func GetInfo(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	log.GroupOp(deviceID, jid, "GetGroupInfo", groupJID).Info("Getting group info")

	// Use WhatsAppComposeJID for group JIDs - WhatsAppGetJID is only for personal phone numbers
	groupID := pkgWhatsApp.WhatsAppComposeJID(groupJID)

	groupInfo, err := pkgWhatsApp.WhatsAppGroupInfo(ctx, jid, deviceID, groupID)
	if err != nil {
		log.GroupOp(deviceID, jid, "GetGroupInfo", groupJID).WithError(err).Error("Failed to get group info")
		return router.ResponseInternalError(c, err.Error())
	}

	log.GroupOp(deviceID, jid, "GetGroupInfo", groupJID).Info("Group info retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get group info", groupInfo)
}

func Create(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	var reqCreate typWhatsApp.RequestCreateGroup
	err := c.BodyParser(&reqCreate)
	if err != nil {
		log.GroupOp(deviceID, jid, "CreateGroup", "").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.GroupOp(deviceID, jid, "CreateGroup", "").WithField("name", reqCreate.Name).WithField("participants", len(reqCreate.Participants)).Info("Creating group")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	groupInfo, err := pkgWhatsApp.WhatsAppCreateGroupEnhanced(ctx, jid, deviceID, reqCreate.Name, reqCreate.Participants, reqCreate.Description, reqCreate.Photo)
	if err != nil {
		log.GroupOp(deviceID, jid, "CreateGroup", "").WithField("name", reqCreate.Name).WithError(err).Error("Failed to create group")
		return router.ResponseInternalError(c, err.Error())
	}

	log.GroupOp(deviceID, jid, "CreateGroup", "").WithField("name", reqCreate.Name).Info("Group created successfully")

	return router.ResponseSuccessWithData(c, "Success create group", groupInfo)
}

func Leave(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	log.GroupOp(deviceID, jid, "LeaveGroup", groupJID).Info("Leaving group")

	// Pass groupJID directly - WhatsAppGroupLeave handles JID parsing internally
	err := pkgWhatsApp.WhatsAppGroupLeave(ctx, jid, deviceID, groupJID)
	if err != nil {
		log.GroupOp(deviceID, jid, "LeaveGroup", groupJID).WithError(err).Error("Failed to leave group")
		return router.ResponseInternalError(c, err.Error())
	}

	log.GroupOp(deviceID, jid, "LeaveGroup", groupJID).Info("Left group successfully")

	return router.ResponseSuccess(c, "Success leave group")
}

func UpdateName(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	var req struct {
		Name string `json:"name"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.GroupOp(deviceID, jid, "UpdateGroupName", groupJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.GroupOp(deviceID, jid, "UpdateGroupName", groupJID).WithField("new_name", req.Name).Info("Updating group name")

	// Use WhatsAppComposeJID for group JIDs - WhatsAppGetJID is only for personal phone numbers
	groupID := pkgWhatsApp.WhatsAppComposeJID(groupJID)

	err = pkgWhatsApp.WhatsAppGroupUpdateName(ctx, jid, deviceID, groupID, req.Name)
	if err != nil {
		log.GroupOp(deviceID, jid, "UpdateGroupName", groupJID).WithError(err).Error("Failed to update group name")
		return router.ResponseInternalError(c, err.Error())
	}

	log.GroupOp(deviceID, jid, "UpdateGroupName", groupJID).WithField("new_name", req.Name).Info("Group name updated successfully")

	return router.ResponseSuccess(c, "Success update group name")
}

func UpdateDescription(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	var req struct {
		Description string `json:"description"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.GroupOp(deviceID, jid, "UpdateGroupDescription", groupJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.GroupOp(deviceID, jid, "UpdateGroupDescription", groupJID).Info("Updating group description")

	// Use WhatsAppComposeJID for group JIDs - WhatsAppGetJID is only for personal phone numbers
	groupID := pkgWhatsApp.WhatsAppComposeJID(groupJID)

	err = pkgWhatsApp.WhatsAppGroupUpdateDescription(ctx, jid, deviceID, groupID, req.Description)
	if err != nil {
		log.GroupOp(deviceID, jid, "UpdateGroupDescription", groupJID).WithError(err).Error("Failed to update group description")
		return router.ResponseInternalError(c, err.Error())
	}

	log.GroupOp(deviceID, jid, "UpdateGroupDescription", groupJID).Info("Group description updated successfully")

	return router.ResponseSuccess(c, "Success update group description")
}

func UpdatePhoto(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	log.GroupOp(deviceID, jid, "UpdateGroupPhoto", groupJID).Info("Updating group photo")

	// Use WhatsAppComposeJID for group JIDs - WhatsAppGetJID is only for personal phone numbers
	groupID := pkgWhatsApp.WhatsAppComposeJID(groupJID)

	fileHeader, err := c.FormFile("photo")
	if err != nil {
		log.GroupOp(deviceID, jid, "UpdateGroupPhoto", groupJID).Warn("No photo provided")
		return router.ResponseBadRequest(c, "photo is required")
	}

	file, err := fileHeader.Open()
	if err != nil {
		log.GroupOp(deviceID, jid, "UpdateGroupPhoto", groupJID).WithError(err).Error("Failed to open photo file")
		return router.ResponseInternalError(c, err.Error())
	}
	defer file.Close()

	photoURL, err := pkgWhatsApp.WhatsAppGroupUpdatePhoto(ctx, jid, deviceID, groupID, file)
	if err != nil {
		log.GroupOp(deviceID, jid, "UpdateGroupPhoto", groupJID).WithError(err).Error("Failed to update group photo")
		return router.ResponseInternalError(c, err.Error())
	}

	log.GroupOp(deviceID, jid, "UpdateGroupPhoto", groupJID).Info("Group photo updated successfully")

	return router.ResponseSuccessWithData(c, "Success update group photo", map[string]interface{}{"photo_url": photoURL})
}

func GetInviteLink(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	reset := c.QueryBool("reset", false)

	log.GroupOp(deviceID, jid, "GetInviteLink", groupJID).WithField("reset", reset).Info("Getting group invite link")

	// Pass groupJID directly - WhatsAppGroupInviteLink handles JID parsing internally
	inviteLink, err := pkgWhatsApp.WhatsAppGroupInviteLink(ctx, jid, deviceID, groupJID, reset)
	if err != nil {
		log.GroupOp(deviceID, jid, "GetInviteLink", groupJID).WithError(err).Error("Failed to get invite link")
		return router.ResponseInternalError(c, err.Error())
	}

	log.GroupOp(deviceID, jid, "GetInviteLink", groupJID).WithField("reset", reset).Info("Invite link retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get invite link", map[string]interface{}{"link": inviteLink})
}

func UpdateSettings(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	var req typWhatsApp.RequestUpdateGroupSettings
	err := c.BodyParser(&req)
	if err != nil {
		log.GroupOp(deviceID, jid, "UpdateSettings", groupJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.GroupOp(deviceID, jid, "UpdateSettings", groupJID).Info("Updating group settings")

	// Use WhatsAppComposeJID for group JIDs - WhatsAppGetJID is only for personal phone numbers
	groupID := pkgWhatsApp.WhatsAppComposeJID(groupJID)

	err = pkgWhatsApp.WhatsAppGroupUpdateSettings(jid, deviceID, groupID, req)
	if err != nil {
		log.GroupOp(deviceID, jid, "UpdateSettings", groupJID).WithError(err).Error("Failed to update group settings")
		return router.ResponseInternalError(c, err.Error())
	}

	log.GroupOp(deviceID, jid, "UpdateSettings", groupJID).Info("Group settings updated successfully")

	return router.ResponseSuccess(c, "Success update group settings")
}

func GetParticipantRequests(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	log.GroupOp(deviceID, jid, "GetParticipantRequests", groupJID).Info("Getting participant requests")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	// Use WhatsAppComposeJID for group JIDs - WhatsAppGetJID is only for personal phone numbers
	groupID := pkgWhatsApp.WhatsAppComposeJID(groupJID)

	requests, err := pkgWhatsApp.WhatsAppGroupParticipantRequests(ctx, jid, deviceID, groupID)
	if err != nil {
		log.GroupOp(deviceID, jid, "GetParticipantRequests", groupJID).WithError(err).Error("Failed to get participant requests")
		return router.ResponseInternalError(c, err.Error())
	}

	requestCount := 0
	if requests != nil {
		requestCount = len(requests)
	}

	log.GroupOp(deviceID, jid, "GetParticipantRequests", groupJID).WithField("request_count", requestCount).Info("Participant requests retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get participant requests", requests)
}

func SetJoinApproval(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	var req struct {
		Mode bool `json:"mode"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.GroupOp(deviceID, jid, "SetJoinApproval", groupJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.GroupOp(deviceID, jid, "SetJoinApproval", groupJID).WithField("mode", req.Mode).Info("Setting join approval mode")

	// Use WhatsAppComposeJID for group JIDs - WhatsAppGetJID is only for personal phone numbers
	groupID := pkgWhatsApp.WhatsAppComposeJID(groupJID)

	err = pkgWhatsApp.WhatsAppGroupJoinApprovalMode(jid, deviceID, groupID, req.Mode)
	if err != nil {
		log.GroupOp(deviceID, jid, "SetJoinApproval", groupJID).WithError(err).Error("Failed to set join approval mode")
		return router.ResponseInternalError(c, err.Error())
	}

	log.GroupOp(deviceID, jid, "SetJoinApproval", groupJID).WithField("mode", req.Mode).Info("Join approval mode set successfully")

	return router.ResponseSuccess(c, "Success set join approval mode")
}

func GetInfoFromInvite(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	inviteCode := c.Params("invite_code")

	log.GroupOp(deviceID, jid, "GetInfoFromInvite", "").WithField("invite_code", inviteCode).Info("Getting group info from invite")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	groupInfo, err := pkgWhatsApp.WhatsAppGroupInfoFromInvite(ctx, jid, deviceID, inviteCode)
	if err != nil {
		log.GroupOp(deviceID, jid, "GetInfoFromInvite", "").WithField("invite_code", inviteCode).WithError(err).Error("Failed to get group info from invite")
		return router.ResponseInternalError(c, err.Error())
	}

	log.GroupOp(deviceID, jid, "GetInfoFromInvite", "").WithField("invite_code", inviteCode).Info("Group info from invite retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get group info", groupInfo)
}

func JoinWithInvite(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	var req typWhatsApp.RequestJoinGroupInvite
	err := c.BodyParser(&req)
	if err != nil {
		log.GroupOp(deviceID, jid, "JoinWithInvite", groupJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.GroupOp(deviceID, jid, "JoinWithInvite", groupJID).WithField("invite_code", req.InviteCode).Info("Joining group with invite")

	// Pass groupJID directly - WhatsAppGroupJoinWithInvite handles JID parsing internally
	err = pkgWhatsApp.WhatsAppGroupJoinWithInvite(jid, deviceID, groupJID, req.Inviter, req.InviteCode, req.Expiration)
	if err != nil {
		log.GroupOp(deviceID, jid, "JoinWithInvite", groupJID).WithError(err).Error("Failed to join group with invite")
		return router.ResponseInternalError(c, err.Error())
	}

	log.GroupOp(deviceID, jid, "JoinWithInvite", groupJID).Info("Joined group with invite successfully")

	return router.ResponseSuccess(c, "Success join group with invite")
}

func SetMemberAddMode(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	var req struct {
		Mode string `json:"mode"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.GroupOp(deviceID, jid, "SetMemberAddMode", groupJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.GroupOp(deviceID, jid, "SetMemberAddMode", groupJID).WithField("mode", req.Mode).Info("Setting member add mode")

	// Pass groupJID directly - WhatsAppGroupSetMemberAddMode handles JID parsing internally
	err = pkgWhatsApp.WhatsAppGroupSetMemberAddMode(jid, deviceID, groupJID, req.Mode)
	if err != nil {
		log.GroupOp(deviceID, jid, "SetMemberAddMode", groupJID).WithError(err).Error("Failed to set member add mode")
		return router.ResponseInternalError(c, err.Error())
	}

	log.GroupOp(deviceID, jid, "SetMemberAddMode", groupJID).WithField("mode", req.Mode).Info("Member add mode set successfully")

	return router.ResponseSuccess(c, "Success set member add mode")
}

func SetTopic(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	var req typWhatsApp.RequestSetGroupTopic
	err := c.BodyParser(&req)
	if err != nil {
		log.GroupOp(deviceID, jid, "SetTopic", groupJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.GroupOp(deviceID, jid, "SetTopic", groupJID).WithField("topic", req.Topic).Info("Setting group topic")

	// Pass groupJID directly - WhatsAppGroupSetTopic handles JID parsing internally
	err = pkgWhatsApp.WhatsAppGroupSetTopic(jid, deviceID, groupJID, req.PreviousID, req.NewID, req.Topic)
	if err != nil {
		log.GroupOp(deviceID, jid, "SetTopic", groupJID).WithError(err).Error("Failed to set group topic")
		return router.ResponseInternalError(c, err.Error())
	}

	log.GroupOp(deviceID, jid, "SetTopic", groupJID).Info("Group topic set successfully")

	return router.ResponseSuccess(c, "Success set group topic")
}

func LinkGroup(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	parentGroupJID := c.Params("parent_group_jid")
	childGroupJID := c.Params("group_jid")

	log.GroupOp(deviceID, jid, "LinkGroup", parentGroupJID).WithField("child_group", childGroupJID).Info("Linking groups")

	// Pass group JIDs directly - WhatsAppGroupLink handles JID parsing internally
	err := pkgWhatsApp.WhatsAppGroupLink(jid, deviceID, parentGroupJID, childGroupJID)
	if err != nil {
		log.GroupOp(deviceID, jid, "LinkGroup", parentGroupJID).WithField("child_group", childGroupJID).WithError(err).Error("Failed to link groups")
		return router.ResponseInternalError(c, err.Error())
	}

	log.GroupOp(deviceID, jid, "LinkGroup", parentGroupJID).WithField("child_group", childGroupJID).Info("Groups linked successfully")

	return router.ResponseSuccess(c, "Success link groups")
}

func GetLinkedParticipants(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	communityJID := c.Params("community_jid")

	log.GroupOp(deviceID, jid, "GetLinkedParticipants", communityJID).Info("Getting linked participants")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	// Pass communityJID directly - WhatsAppGroupGetLinkedParticipants handles JID parsing internally
	participants, err := pkgWhatsApp.WhatsAppGroupGetLinkedParticipants(ctx, jid, deviceID, communityJID)
	if err != nil {
		log.GroupOp(deviceID, jid, "GetLinkedParticipants", communityJID).WithError(err).Error("Failed to get linked participants")
		return router.ResponseInternalError(c, err.Error())
	}

	participantCount := 0
	if participants != nil {
		participantCount = len(participants)
	}

	log.GroupOp(deviceID, jid, "GetLinkedParticipants", communityJID).WithField("participant_count", participantCount).Info("Linked participants retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get linked participants", participants)
}

func GetSubGroups(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	communityJID := c.Params("community_jid")

	log.GroupOp(deviceID, jid, "GetSubGroups", communityJID).Info("Getting sub groups")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	// Pass communityJID directly - WhatsAppGroupGetSubGroups handles JID parsing internally
	subGroups, err := pkgWhatsApp.WhatsAppGroupGetSubGroups(ctx, jid, deviceID, communityJID)
	if err != nil {
		log.GroupOp(deviceID, jid, "GetSubGroups", communityJID).WithError(err).Error("Failed to get sub groups")
		return router.ResponseInternalError(c, err.Error())
	}

	subGroupCount := 0
	if subGroups != nil {
		subGroupCount = len(subGroups)
	}

	log.GroupOp(deviceID, jid, "GetSubGroups", communityJID).WithField("sub_group_count", subGroupCount).Info("Sub groups retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get sub groups", subGroups)
}

func AddParticipants(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	var req struct {
		Participants []string `json:"participants"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.GroupOp(deviceID, jid, "AddParticipants", groupJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.GroupOp(deviceID, jid, "AddParticipants", groupJID).WithField("participant_count", len(req.Participants)).Info("Adding participants to group")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	// Pass groupJID directly - WhatsAppAddParticipants handles JID parsing internally
	participants, err := pkgWhatsApp.WhatsAppAddParticipants(ctx, jid, deviceID, groupJID, req.Participants)
	if err != nil {
		log.GroupOp(deviceID, jid, "AddParticipants", groupJID).WithError(err).Error("Failed to add participants")
		return router.ResponseInternalError(c, err.Error())
	}

	log.GroupOp(deviceID, jid, "AddParticipants", groupJID).WithField("participant_count", len(req.Participants)).Info("Participants added successfully")

	return router.ResponseSuccessWithData(c, "Success add participants", participants)
}

func RemoveParticipants(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	var req struct {
		Participants []string `json:"participants"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.GroupOp(deviceID, jid, "RemoveParticipants", groupJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.GroupOp(deviceID, jid, "RemoveParticipants", groupJID).WithField("participant_count", len(req.Participants)).Info("Removing participants from group")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	// Pass groupJID directly - WhatsAppRemoveParticipants handles JID parsing internally
	participants, err := pkgWhatsApp.WhatsAppRemoveParticipants(ctx, jid, deviceID, groupJID, req.Participants)
	if err != nil {
		log.GroupOp(deviceID, jid, "RemoveParticipants", groupJID).WithError(err).Error("Failed to remove participants")
		return router.ResponseInternalError(c, err.Error())
	}

	log.GroupOp(deviceID, jid, "RemoveParticipants", groupJID).WithField("participant_count", len(req.Participants)).Info("Participants removed successfully")

	return router.ResponseSuccessWithData(c, "Success remove participants", participants)
}

func ApproveRequests(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	var req struct {
		Users []string `json:"users"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.GroupOp(deviceID, jid, "ApproveRequests", groupJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.GroupOp(deviceID, jid, "ApproveRequests", groupJID).WithField("user_count", len(req.Users)).Info("Approving join requests")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	// Pass groupJID directly - WhatsAppApproveJoinRequests handles JID parsing internally
	participants, err := pkgWhatsApp.WhatsAppApproveJoinRequests(ctx, jid, deviceID, groupJID, req.Users)
	if err != nil {
		log.GroupOp(deviceID, jid, "ApproveRequests", groupJID).WithError(err).Error("Failed to approve join requests")
		return router.ResponseInternalError(c, err.Error())
	}

	log.GroupOp(deviceID, jid, "ApproveRequests", groupJID).WithField("user_count", len(req.Users)).Info("Join requests approved successfully")

	return router.ResponseSuccessWithData(c, "Success approve join requests", participants)
}

func RejectRequests(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	var req struct {
		Users []string `json:"users"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.GroupOp(deviceID, jid, "RejectRequests", groupJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.GroupOp(deviceID, jid, "RejectRequests", groupJID).WithField("user_count", len(req.Users)).Info("Rejecting join requests")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	// Pass groupJID directly - WhatsAppRejectJoinRequests handles JID parsing internally
	participants, err := pkgWhatsApp.WhatsAppRejectJoinRequests(ctx, jid, deviceID, groupJID, req.Users)
	if err != nil {
		log.GroupOp(deviceID, jid, "RejectRequests", groupJID).WithError(err).Error("Failed to reject join requests")
		return router.ResponseInternalError(c, err.Error())
	}

	log.GroupOp(deviceID, jid, "RejectRequests", groupJID).WithField("user_count", len(req.Users)).Info("Join requests rejected successfully")

	return router.ResponseSuccessWithData(c, "Success reject join requests", participants)
}

func PromoteAdmins(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	var req struct {
		Users []string `json:"users"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.GroupOp(deviceID, jid, "PromoteAdmins", groupJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.GroupOp(deviceID, jid, "PromoteAdmins", groupJID).WithField("user_count", len(req.Users)).Info("Promoting users to admin")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	// Pass groupJID directly - WhatsAppPromoteAdmins handles JID parsing internally
	participants, err := pkgWhatsApp.WhatsAppPromoteAdmins(ctx, jid, deviceID, groupJID, req.Users)
	if err != nil {
		log.GroupOp(deviceID, jid, "PromoteAdmins", groupJID).WithError(err).Error("Failed to promote admins")
		return router.ResponseInternalError(c, err.Error())
	}

	log.GroupOp(deviceID, jid, "PromoteAdmins", groupJID).WithField("user_count", len(req.Users)).Info("Users promoted to admin successfully")

	return router.ResponseSuccessWithData(c, "Success promote admins", participants)
}

func DemoteAdmins(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	var req struct {
		Users []string `json:"users"`
	}
	err := c.BodyParser(&req)
	if err != nil {
		log.GroupOp(deviceID, jid, "DemoteAdmins", groupJID).Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.GroupOp(deviceID, jid, "DemoteAdmins", groupJID).WithField("user_count", len(req.Users)).Info("Demoting admins")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	// Pass groupJID directly - WhatsAppDemoteAdmins handles JID parsing internally
	participants, err := pkgWhatsApp.WhatsAppDemoteAdmins(ctx, jid, deviceID, groupJID, req.Users)
	if err != nil {
		log.GroupOp(deviceID, jid, "DemoteAdmins", groupJID).WithError(err).Error("Failed to demote admins")
		return router.ResponseInternalError(c, err.Error())
	}

	log.GroupOp(deviceID, jid, "DemoteAdmins", groupJID).WithField("user_count", len(req.Users)).Info("Admins demoted successfully")

	return router.ResponseSuccessWithData(c, "Success demote admins", participants)
}

// UnlinkGroup unlinks a subgroup from a community/parent group
func UnlinkGroup(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	parentJID := c.Params("parent_jid")
	childJID := c.Params("child_jid")

	log.GroupOp(deviceID, jid, "UnlinkGroup", parentJID).
		WithField("child_jid", childJID).
		Info("Unlinking subgroup from community")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	err := pkgWhatsApp.WhatsAppUnlinkGroup(ctx, jid, deviceID, parentJID, childJID)
	if err != nil {
		log.GroupOp(deviceID, jid, "UnlinkGroup", parentJID).
			WithField("child_jid", childJID).
			WithError(err).
			Error("Failed to unlink subgroup")
		return router.ResponseInternalError(c, err.Error())
	}

	log.GroupOp(deviceID, jid, "UnlinkGroup", parentJID).
		WithField("child_jid", childJID).
		Info("Subgroup unlinked successfully")

	return router.ResponseSuccess(c, "Success unlink subgroup from community")
}
