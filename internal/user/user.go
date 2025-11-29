package user

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

func GetInfo(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	userJID := c.Params("user_jid")

	var reqUserInfo typWhatsApp.RequestUserInfo
	reqUserInfo.Phone = userJID

	userInfo, err := pkgWhatsApp.WhatsAppGetUserInfo(ctx, jid, deviceID, []string{userJID})
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	var resUserInfo typWhatsApp.ResponseUserInfo
	if info, exists := userInfo[userJID]; exists {
		resUserInfo.Status = info.Status
		resUserInfo.PictureID = info.PictureID
	}

	return router.ResponseSuccessWithData(c, "Success get user info", resUserInfo)
}

func GetProfilePicture(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	userJID := c.Params("user_jid")

	pictureInfo, err := pkgWhatsApp.WhatsAppGetUserProfilePicture(ctx, jid, deviceID, userJID, false)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	var resUserPicture typWhatsApp.ResponseUserPicture
	resUserPicture.URL = pictureInfo.URL
	resUserPicture.ID = pictureInfo.ID
	resUserPicture.Type = pictureInfo.Type
	resUserPicture.DirectURL = pictureInfo.DirectPath

	return router.ResponseSuccessWithData(c, "Success get user picture", resUserPicture)
}

func BlockUser(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	userJID := c.Params("user_jid")

	err := pkgWhatsApp.WhatsAppBlockUser(ctx, jid, deviceID, userJID)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccess(c, "Success block user")
}

func UnblockUser(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	userJID := c.Params("user_jid")

	err := pkgWhatsApp.WhatsAppUnblockUser(ctx, jid, deviceID, userJID)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccess(c, "Success unblock user")
}

func GetPrivacy(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	privacy, err := pkgWhatsApp.WhatsAppGetPrivacy(ctx, jid, deviceID)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Success get privacy settings", privacy)
}

func UpdatePrivacy(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	var reqPrivacy typWhatsApp.RequestPrivacy
	err := c.BodyParser(&reqPrivacy)
	if err != nil {
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	privacy, err := pkgWhatsApp.WhatsAppSetUserPrivacy(ctx, jid, deviceID, reqPrivacy.Setting, reqPrivacy.Value)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Success update privacy settings", privacy)
}

func GetStatusPrivacy(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	statusPrivacy, err := pkgWhatsApp.WhatsAppGetStatusPrivacy(ctx, jid, deviceID)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Success get status privacy", statusPrivacy)
}

func UpdateStatus(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	var reqStatus typWhatsApp.RequestStatus
	err := c.BodyParser(&reqStatus)
	if err != nil {
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	err = pkgWhatsApp.WhatsAppSetUserStatus(jid, deviceID, reqStatus.Status)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccess(c, "Success update status")
}

func GetDevices(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	userJID := c.Params("jid")

	phoneJID := pkgWhatsApp.WhatsAppGetJID(ctx, jid, deviceID, userJID)

	devices, err := pkgWhatsApp.WhatsAppGetUserDevices(ctx, jid, deviceID, phoneJID)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Success get user devices", devices)
}
