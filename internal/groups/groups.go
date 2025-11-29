package groups

import (
	"context"

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

func List(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	groups, err := pkgWhatsApp.WhatsAppGroupGet(jid, deviceID)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Success get groups", groups)
}

func GetInfo(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	groupID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, groupJID)

	groupInfo, err := pkgWhatsApp.WhatsAppGroupInfo(ctx, jid, deviceID, groupID)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Success get group info", groupInfo)
}

func Create(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	var reqCreate typWhatsApp.RequestCreateGroup
	err := c.BodyParser(&reqCreate)
	if err != nil {
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	groupInfo, err := pkgWhatsApp.WhatsAppCreateGroupEnhanced(ctx, jid, deviceID, reqCreate.Name, reqCreate.Participants, reqCreate.Description, reqCreate.Photo)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Success create group", groupInfo)
}

func Leave(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	groupID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, groupJID)

	err := pkgWhatsApp.WhatsAppGroupLeave(ctx, jid, deviceID, groupID.String())
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

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
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	groupID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, groupJID)

	err = pkgWhatsApp.WhatsAppGroupUpdateName(ctx, jid, deviceID, groupID, req.Name)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

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
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	groupID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, groupJID)

	err = pkgWhatsApp.WhatsAppGroupUpdateDescription(ctx, jid, deviceID, groupID, req.Description)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccess(c, "Success update group description")
}

func UpdatePhoto(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	groupID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, groupJID)

	fileHeader, err := c.FormFile("photo")
	if err != nil {
		return router.ResponseBadRequest(c, "photo is required")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}
	defer file.Close()

	photoURL, err := pkgWhatsApp.WhatsAppGroupUpdatePhoto(ctx, jid, deviceID, groupID, file)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

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

	groupID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, groupJID)

	inviteLink, err := pkgWhatsApp.WhatsAppGroupInviteLink(ctx, jid, deviceID, groupID.String(), reset)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Success get invite link", map[string]interface{}{"link": inviteLink})
}

func UpdateSettings(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	var req typWhatsApp.RequestUpdateGroupSettings
	err := c.BodyParser(&req)
	if err != nil {
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	groupID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, groupJID)

	err = pkgWhatsApp.WhatsAppGroupUpdateSettings(jid, deviceID, groupID, req)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccess(c, "Success update group settings")
}

func GetParticipantRequests(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	groupID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, groupJID)

	requests, err := pkgWhatsApp.WhatsAppGroupParticipantRequests(ctx, jid, deviceID, groupID)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

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
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	groupID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, groupJID)

	err = pkgWhatsApp.WhatsAppGroupJoinApprovalMode(jid, deviceID, groupID, req.Mode)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccess(c, "Success set join approval mode")
}

func GetInfoFromInvite(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	inviteCode := c.Params("invite_code")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	groupInfo, err := pkgWhatsApp.WhatsAppGroupInfoFromInvite(ctx, jid, deviceID, inviteCode)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Success get group info", groupInfo)
}

func JoinWithInvite(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	var req typWhatsApp.RequestJoinGroupInvite
	err := c.BodyParser(&req)
	if err != nil {
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	groupID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, groupJID)

	err = pkgWhatsApp.WhatsAppGroupJoinWithInvite(jid, deviceID, groupID.String(), req.Inviter, req.InviteCode, req.Expiration)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

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
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	groupID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, groupJID)

	err = pkgWhatsApp.WhatsAppGroupSetMemberAddMode(jid, deviceID, groupID.String(), req.Mode)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccess(c, "Success set member add mode")
}

func SetTopic(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	groupJID := c.Params("group_jid")

	var req typWhatsApp.RequestSetGroupTopic
	err := c.BodyParser(&req)
	if err != nil {
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	groupID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, groupJID)

	err = pkgWhatsApp.WhatsAppGroupSetTopic(jid, deviceID, groupID.String(), req.PreviousID, req.NewID, req.Topic)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccess(c, "Success set group topic")
}

func LinkGroup(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	parentGroupJID := c.Params("parent_group_jid")
	childGroupJID := c.Params("group_jid")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	parentID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, parentGroupJID)
	childID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, childGroupJID)

	err := pkgWhatsApp.WhatsAppGroupLink(jid, deviceID, parentID.String(), childID.String())
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccess(c, "Success link groups")
}

func GetLinkedParticipants(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	communityJID := c.Params("community_jid")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	communityID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, communityJID)

	participants, err := pkgWhatsApp.WhatsAppGroupGetLinkedParticipants(ctx, jid, deviceID, communityID.String())
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Success get linked participants", participants)
}

func GetSubGroups(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)
	communityJID := c.Params("community_jid")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	communityID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, communityJID)

	subGroups, err := pkgWhatsApp.WhatsAppGroupGetSubGroups(ctx, jid, deviceID, communityID.String())
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

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
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	groupID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, groupJID)

	participants, err := pkgWhatsApp.WhatsAppAddParticipants(ctx, jid, deviceID, groupID.String(), req.Participants)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

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
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	groupID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, groupJID)

	participants, err := pkgWhatsApp.WhatsAppRemoveParticipants(ctx, jid, deviceID, groupID.String(), req.Participants)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

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
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	groupID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, groupJID)

	participants, err := pkgWhatsApp.WhatsAppApproveJoinRequests(ctx, jid, deviceID, groupID.String(), req.Users)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

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
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	groupID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, groupJID)

	participants, err := pkgWhatsApp.WhatsAppRejectJoinRequests(ctx, jid, deviceID, groupID.String(), req.Users)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

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
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	groupID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, groupJID)

	participants, err := pkgWhatsApp.WhatsAppPromoteAdmins(ctx, jid, deviceID, groupID.String(), req.Users)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

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
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	groupID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, groupJID)

	participants, err := pkgWhatsApp.WhatsAppDemoteAdmins(ctx, jid, deviceID, groupID.String(), req.Users)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Success demote admins", participants)
}

func GetJoined(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	groups, err := pkgWhatsApp.WhatsAppGroupGet(jid, deviceID)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Success get joined groups", groups)
}
