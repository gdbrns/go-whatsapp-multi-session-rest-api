package user

import (
	"context"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/log"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/router"
	pkgWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/whatsapp"
	typWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/types"
)

// decodeUserJID URL-decodes the user JID parameter from the route
func decodeUserJID(encoded string) string {
	decoded, err := url.PathUnescape(encoded)
	if err != nil {
		return encoded
	}
	return decoded
}

// normalizeToUserJID converts a device JID (e.g., "6281378887612:74@s.whatsapp.net")
// to a base user JID (e.g., "6281378887612@s.whatsapp.net") by removing the device part
func normalizeToUserJID(jidStr string) string {
	// If there's no @ symbol, it's already a phone number, return as-is
	if !strings.Contains(jidStr, "@") {
		return jidStr
	}
	
	// Split at @ to get user part and server
	parts := strings.SplitN(jidStr, "@", 2)
	if len(parts) != 2 {
		return jidStr
	}
	
	userPart := parts[0]
	server := parts[1]
	
	// If user part contains ":" (device ID separator), remove it
	// e.g., "6281378887612:74" -> "6281378887612"
	if colonIdx := strings.Index(userPart, ":"); colonIdx != -1 {
		userPart = userPart[:colonIdx]
	}
	
	return userPart + "@" + server
}

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
	rawJID := decodeUserJID(c.Params("user_jid"))
	userJID := normalizeToUserJID(rawJID)

	log.Session(c, "GetUserInfo").WithField("target_user", userJID).Info("Getting user info")

	var reqUserInfo typWhatsApp.RequestUserInfo
	reqUserInfo.Phone = userJID

	userInfo, err := pkgWhatsApp.WhatsAppGetUserInfo(ctx, jid, deviceID, []string{userJID})
	if err != nil {
		log.Session(c, "GetUserInfo").WithField("target_user", userJID).WithError(err).Error("Failed to get user info")
		return router.ResponseInternalError(c, err.Error())
	}

	var resUserInfo typWhatsApp.ResponseUserInfo
	if info, exists := userInfo[userJID]; exists {
		resUserInfo.Status = info.Status
		resUserInfo.PictureID = info.PictureID
	}

	log.Session(c, "GetUserInfo").WithField("target_user", userJID).Info("User info retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get user info", resUserInfo)
}

func GetProfilePicture(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	rawJID := decodeUserJID(c.Params("user_jid"))
	userJID := normalizeToUserJID(rawJID)

	log.Session(c, "GetProfilePicture").WithField("target_user", userJID).Info("Getting user profile picture")

	pictureInfo, err := pkgWhatsApp.WhatsAppGetUserProfilePicture(ctx, jid, deviceID, userJID, false)
	if err != nil {
		log.Session(c, "GetProfilePicture").WithField("target_user", userJID).WithError(err).Error("Failed to get profile picture")
		return router.ResponseInternalError(c, err.Error())
	}

	var resUserPicture typWhatsApp.ResponseUserPicture
	resUserPicture.URL = pictureInfo.URL
	resUserPicture.ID = pictureInfo.ID
	resUserPicture.Type = pictureInfo.Type
	resUserPicture.DirectURL = pictureInfo.DirectPath

	log.Session(c, "GetProfilePicture").WithField("target_user", userJID).Info("Profile picture retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get user picture", resUserPicture)
}

func BlockUser(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	userJID := decodeUserJID(c.Params("user_jid"))

	log.Session(c, "BlockUser").WithField("target_user", userJID).Info("Blocking user")

	err := pkgWhatsApp.WhatsAppBlockUser(ctx, jid, deviceID, userJID)
	if err != nil {
		log.Session(c, "BlockUser").WithField("target_user", userJID).WithError(err).Error("Failed to block user")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Session(c, "BlockUser").WithField("target_user", userJID).Info("User blocked successfully")

	return router.ResponseSuccess(c, "Success block user")
}

func UnblockUser(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	userJID := decodeUserJID(c.Params("user_jid"))

	log.Session(c, "UnblockUser").WithField("target_user", userJID).Info("Unblocking user")

	err := pkgWhatsApp.WhatsAppUnblockUser(ctx, jid, deviceID, userJID)
	if err != nil {
		log.Session(c, "UnblockUser").WithField("target_user", userJID).WithError(err).Error("Failed to unblock user")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Session(c, "UnblockUser").WithField("target_user", userJID).Info("User unblocked successfully")

	return router.ResponseSuccess(c, "Success unblock user")
}

func GetPrivacy(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	log.Session(c, "GetPrivacy").Info("Getting privacy settings")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	privacy, err := pkgWhatsApp.WhatsAppGetPrivacy(ctx, jid, deviceID)
	if err != nil {
		log.Session(c, "GetPrivacy").WithError(err).Error("Failed to get privacy settings")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Session(c, "GetPrivacy").Info("Privacy settings retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get privacy settings", privacy)
}

func UpdatePrivacy(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	var reqPrivacy typWhatsApp.RequestPrivacy
	err := c.BodyParser(&reqPrivacy)
	if err != nil {
		log.Session(c, "UpdatePrivacy").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.Session(c, "UpdatePrivacy").WithField("setting", reqPrivacy.Setting).WithField("value", reqPrivacy.Value).Info("Updating privacy setting")

	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	privacy, err := pkgWhatsApp.WhatsAppSetUserPrivacy(ctx, jid, deviceID, reqPrivacy.Setting, reqPrivacy.Value)
	if err != nil {
		log.Session(c, "UpdatePrivacy").WithField("setting", reqPrivacy.Setting).WithError(err).Error("Failed to update privacy setting")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Session(c, "UpdatePrivacy").WithField("setting", reqPrivacy.Setting).Info("Privacy setting updated successfully")

	return router.ResponseSuccessWithData(c, "Success update privacy settings", privacy)
}

func GetStatusPrivacy(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	log.Session(c, "GetStatusPrivacy").Info("Getting status privacy settings")

	statusPrivacy, err := pkgWhatsApp.WhatsAppGetStatusPrivacy(ctx, jid, deviceID)
	if err != nil {
		log.Session(c, "GetStatusPrivacy").WithError(err).Error("Failed to get status privacy")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Session(c, "GetStatusPrivacy").Info("Status privacy retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get status privacy", statusPrivacy)
}

func UpdateStatus(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	var reqStatus typWhatsApp.RequestStatus
	err := c.BodyParser(&reqStatus)
	if err != nil {
		log.Session(c, "UpdateStatus").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	log.Session(c, "UpdateStatus").WithField("status_length", len(reqStatus.Status)).Info("Updating user status")

	err = pkgWhatsApp.WhatsAppSetUserStatus(jid, deviceID, reqStatus.Status)
	if err != nil {
		log.Session(c, "UpdateStatus").WithError(err).Error("Failed to update status")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Session(c, "UpdateStatus").Info("Status updated successfully")

	return router.ResponseSuccess(c, "Success update status")
}

func GetDevices(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	rawJID := decodeUserJID(c.Params("jid"))
	userJID := normalizeToUserJID(rawJID)

	log.Session(c, "GetDevices").WithField("target_user", userJID).Info("Getting user devices")

	// Use WhatsAppComposeJID to parse the normalized JID directly
	phoneJID := pkgWhatsApp.WhatsAppComposeJID(userJID)

	devices, err := pkgWhatsApp.WhatsAppGetUserDevices(ctx, jid, deviceID, phoneJID)
	if err != nil {
		log.Session(c, "GetDevices").WithField("target_user", userJID).WithError(err).Error("Failed to get user devices")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Session(c, "GetDevices").WithField("target_user", userJID).WithField("device_count", len(devices)).Info("User devices retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get user devices", devices)
}
