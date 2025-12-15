package user

import (
	"context"
	"encoding/base64"
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

// SetProfilePhoto sets the current user's profile photo
func SetProfilePhoto(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	log.Session(c, "SetProfilePhoto").Info("Setting profile photo")

	// Try to get photo from form file first
	fileHeader, err := c.FormFile("file")
	var photoBytes []byte

	if err == nil && fileHeader != nil {
		file, err := fileHeader.Open()
		if err != nil {
			log.Session(c, "SetProfilePhoto").WithError(err).Error("Failed to open file")
			return router.ResponseInternalError(c, "Failed to open file")
		}
		defer file.Close()

		photoBytes = make([]byte, fileHeader.Size)
		_, err = file.Read(photoBytes)
		if err != nil {
			log.Session(c, "SetProfilePhoto").WithError(err).Error("Failed to read file")
			return router.ResponseInternalError(c, "Failed to read file")
		}
	} else {
		// Try to get photo from JSON body (base64 encoded)
		var reqPhoto typWhatsApp.RequestSetProfilePhoto
		if err := c.BodyParser(&reqPhoto); err == nil && reqPhoto.PhotoBase64 != "" {
			// Decode base64
			photoBytes, err = base64.StdEncoding.DecodeString(reqPhoto.PhotoBase64)
			if err != nil {
				log.Session(c, "SetProfilePhoto").WithError(err).Error("Failed to decode base64 photo")
				return router.ResponseBadRequest(c, "Invalid base64 photo data")
			}
		} else {
			log.Session(c, "SetProfilePhoto").Warn("No photo provided")
			return router.ResponseBadRequest(c, "Photo file or photo_base64 is required")
		}
	}

	if len(photoBytes) == 0 {
		log.Session(c, "SetProfilePhoto").Warn("Empty photo data")
		return router.ResponseBadRequest(c, "Photo data cannot be empty")
	}

	pictureID, err := pkgWhatsApp.WhatsAppSetProfilePhoto(ctx, jid, deviceID, photoBytes)
	if err != nil {
		log.Session(c, "SetProfilePhoto").WithError(err).Error("Failed to set profile photo")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Session(c, "SetProfilePhoto").WithField("picture_id", pictureID).Info("Profile photo set successfully")

	return router.ResponseSuccessWithData(c, "Success set profile photo", map[string]interface{}{
		"picture_id": pictureID,
	})
}

// ContactSync checks which phone numbers are registered on WhatsApp
func ContactSync(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	log.Session(c, "ContactSync").Info("Syncing contacts")

	var reqSync typWhatsApp.RequestContactSync
	if err := c.BodyParser(&reqSync); err != nil {
		log.Session(c, "ContactSync").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	if len(reqSync.Phones) == 0 {
		log.Session(c, "ContactSync").Warn("Empty phones list")
		return router.ResponseBadRequest(c, "phones list is required and cannot be empty")
	}

	results, err := pkgWhatsApp.WhatsAppContactSync(ctx, jid, deviceID, reqSync.Phones)
	if err != nil {
		log.Session(c, "ContactSync").WithError(err).Error("Failed to sync contacts")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Session(c, "ContactSync").WithField("count", len(results)).Info("Contacts synced successfully")

	return router.ResponseSuccessWithData(c, "Success sync contacts", results)
}

// GetContacts retrieves all saved contacts
func GetContacts(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	log.Session(c, "GetContacts").Info("Getting contacts")

	contacts, err := pkgWhatsApp.WhatsAppGetContacts(ctx, jid, deviceID)
	if err != nil {
		log.Session(c, "GetContacts").WithError(err).Error("Failed to get contacts")
		return router.ResponseInternalError(c, err.Error())
	}

	// Convert to a more usable format
	type contactResponse struct {
		JID          string `json:"jid"`
		PushName     string `json:"push_name"`
		BusinessName string `json:"business_name"`
		FullName     string `json:"full_name"`
		FirstName    string `json:"first_name"`
	}

	result := make([]contactResponse, 0, len(contacts))
	for jidKey, info := range contacts {
		result = append(result, contactResponse{
			JID:          jidKey.String(),
			PushName:     info.PushName,
			BusinessName: info.BusinessName,
			FullName:     info.FullName,
			FirstName:    info.FirstName,
		})
	}

	log.Session(c, "GetContacts").WithField("count", len(result)).Info("Contacts retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get contacts", result)
}

// GetBlocklist retrieves the current user's blocklist
func GetBlocklist(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	log.Session(c, "GetBlocklist").Info("Getting blocklist")

	blocklist, err := pkgWhatsApp.WhatsAppGetBlocklist(ctx, jid, deviceID)
	if err != nil {
		log.Session(c, "GetBlocklist").WithError(err).Error("Failed to get blocklist")
		return router.ResponseInternalError(c, err.Error())
	}

	// Convert to string array
	blockedJIDs := make([]string, len(blocklist))
	for i, jidVal := range blocklist {
		blockedJIDs[i] = jidVal.String()
	}

	log.Session(c, "GetBlocklist").WithField("count", len(blockedJIDs)).Info("Blocklist retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get blocklist", map[string]interface{}{
		"blocked": blockedJIDs,
	})
}

// GetContactQRLink gets the current user's contact QR link
func GetContactQRLink(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	// Check if revoke parameter is set
	revoke := c.QueryBool("revoke", false)

	log.Session(c, "GetContactQRLink").WithField("revoke", revoke).Info("Getting contact QR link")

	link, err := pkgWhatsApp.WhatsAppGetContactQRLink(ctx, jid, deviceID, revoke)
	if err != nil {
		log.Session(c, "GetContactQRLink").WithError(err).Error("Failed to get contact QR link")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Session(c, "GetContactQRLink").Info("Contact QR link retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get contact QR link", map[string]interface{}{
		"link": link,
	})
}

// ResolveContactQRLink resolves a contact QR link code
func ResolveContactQRLink(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	code := c.Params("code")

	if code == "" {
		log.Session(c, "ResolveContactQRLink").Warn("Missing code parameter")
		return router.ResponseBadRequest(c, "code parameter is required")
	}

	log.Session(c, "ResolveContactQRLink").WithField("code", code).Info("Resolving contact QR link")

	target, err := pkgWhatsApp.WhatsAppResolveContactQRLink(ctx, jid, deviceID, code)
	if err != nil {
		log.Session(c, "ResolveContactQRLink").WithField("code", code).WithError(err).Error("Failed to resolve contact QR link")
		return router.ResponseInternalError(c, err.Error())
	}

	if target == nil {
		return router.ResponseNotFound(c, "Contact QR link not found")
	}

	response := map[string]interface{}{
		"jid":       target.JID.String(),
		"type":      target.Type,
		"push_name": target.PushName,
	}

	log.Session(c, "ResolveContactQRLink").WithField("code", code).WithField("target_jid", target.JID.String()).Info("Contact QR link resolved successfully")

	return router.ResponseSuccessWithData(c, "Success resolve contact QR link", response)
}