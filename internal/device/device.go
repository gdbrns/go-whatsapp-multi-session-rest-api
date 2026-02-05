package device

import (
	"context"
	"encoding/base64"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	typWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/types"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/log"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/router"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/validation"
	pkgWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/whatsapp"
	"go.mau.fi/whatsmeow"
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

func ListDevices(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	log.Session(c, "ListDevices").Info("Listing all device routings")

	routings, err := pkgWhatsApp.ListDeviceRoutings(ctx)
	if err != nil {
		log.Session(c, "ListDevices").WithError(err).Error("Failed to list device routings")
		return router.ResponseInternalError(c, err.Error())
	}

	log.Session(c, "ListDevices").WithField("count", len(routings)).Info("Device routings listed successfully")

	return router.ResponseSuccessWithData(c, "Success get device list", routings)
}

// Login initiates QR code login for a device
// Uses X-Device-ID and X-Device-Secret headers for authentication
func Login(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	log.DeviceOpCtx(c, "Login").Info("Initiating QR code login")

	var reqLogin typWhatsApp.RequestLogin
	reqLogin.Output = strings.TrimSpace(c.FormValue("output"))
	if len(reqLogin.Output) == 0 {
		reqLogin.Output = "html"
	}

	pkgWhatsApp.WhatsAppInitClient(nil, jid, deviceID)

	qrCodeImage, qrCodeTimeout, err := pkgWhatsApp.WhatsAppLogin(jid, deviceID)
	if err != nil {
			log.DeviceOpCtx(c, "Login").WithError(err).Error("Failed to generate QR code")
		return router.ResponseInternalError(c, err.Error())
	}

	if qrCodeImage == "WhatsApp Client is Reconnected" {
			log.DeviceOpCtx(c, "Login").Info("WhatsApp client reconnected successfully")
		return router.ResponseSuccess(c, qrCodeImage)
	}

	var resLogin typWhatsApp.ResponseLogin
	resLogin.QRCode = qrCodeImage
	resLogin.Timeout = qrCodeTimeout

	log.DeviceOpCtx(c, "Login").WithField("timeout", qrCodeTimeout).WithField("output", reqLogin.Output).Info("QR code generated successfully")

	if reqLogin.Output == "html" {
		htmlContent := `
		<html>
			<head>
				<title>WhatsApp Multi-Device Login</title>
				<meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no" />
			</head>
			<body>
				<img src="` + resLogin.QRCode + `" />
				<p>
					<b>QR Code Scan</b>
					<br/>
					Timeout in ` + strconv.Itoa(resLogin.Timeout) + ` Second(s)
				</p>
			</body>
		</html>
		`

		c.Set("Content-Type", "text/html")
		return c.SendString(htmlContent)
	}

	return router.ResponseSuccessWithData(c, "Success Generate QR Code", resLogin)
}

// LoginWithCode initiates pairing code login for a device
func LoginWithCode(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	var reqLoginCode typWhatsApp.RequestLoginCode
	err := c.BodyParser(&reqLoginCode)
	if err != nil {
		log.DeviceOpCtx(c, "LoginWithCode").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	phone := strings.TrimSpace(reqLoginCode.Phone)
	if phone == "" {
		log.DeviceOpCtx(c, "LoginWithCode").Warn("Phone number not provided")
		return router.ResponseBadRequest(c, "Phone is required")
	}
	if err := validation.ValidatePhone(phone); err != nil {
		log.DeviceOpCtx(c, "LoginWithCode").WithField("phone", phone).Warn("Invalid phone format")
		return router.ResponseBadRequest(c, err.Error())
	}

	log.DeviceOpCtx(c, "LoginWithCode").WithField("phone", phone).Info("Initiating pairing code login")

	pkgWhatsApp.WhatsAppInitClient(nil, jid, deviceID)

	pairCode, timeout, err := pkgWhatsApp.WhatsAppLoginPair(jid, deviceID, phone)
	if err != nil {
		log.DeviceOpCtx(c, "LoginWithCode").WithField("phone", phone).WithError(err).Error("Failed to generate pairing code")
		return router.ResponseInternalError(c, err.Error())
	}

	var resLoginCode typWhatsApp.ResponseLoginCode
	resLoginCode.PairCode = pairCode
	resLoginCode.Timeout = timeout

	log.DeviceOpCtx(c, "LoginWithCode").WithField("phone", phone).WithField("timeout", timeout).Info("Pairing code generated successfully")

	return router.ResponseSuccessWithData(c, "Success Generate Pairing Code", resLoginCode)
}

// Logout logs out the device from WhatsApp
func Logout(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	log.DeviceOpCtx(c, "Logout").Info("Logging out device")

	err := pkgWhatsApp.WhatsAppLogout(jid, deviceID)
	if err != nil {
		log.DeviceOpCtx(c, "Logout").WithError(err).Error("Failed to logout device")
		return router.ResponseInternalError(c, err.Error())
	}

	log.DeviceOpCtx(c, "Logout").Info("Device logged out successfully")

	return router.ResponseSuccess(c, "Success logout device")
}

// CheckRegistered checks if a phone number is registered on WhatsApp
func CheckRegistered(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)
	phone := c.Params("phone")

	if err := validation.ValidatePhone(phone); err != nil {
		log.DeviceOpCtx(c, "CheckRegistered").WithField("phone", phone).Warn("Invalid phone format")
		return router.ResponseBadRequest(c, err.Error())
	}

	log.DeviceOpCtx(c, "CheckRegistered").WithField("phone", phone).Info("Checking if phone is registered")

	var reqCheckPhone typWhatsApp.RequestCheckPhone
	reqCheckPhone.Phone = phone

	resolvedJID, err := pkgWhatsApp.WhatsAppCheckJID(ctx, jid, deviceID, reqCheckPhone.Phone)

	var resCheckPhone typWhatsApp.ResponseCheckPhone
	resCheckPhone.IsRegistered = err == nil
	if resCheckPhone.IsRegistered && !resolvedJID.IsEmpty() {
		resCheckPhone.JID = resolvedJID.String()
	}

	log.DeviceOpCtx(c, "CheckRegistered").WithField("phone", phone).WithField("is_registered", resCheckPhone.IsRegistered).WithField("jid", resCheckPhone.JID).Info("Phone registration check complete")

	return router.ResponseSuccessWithData(c, "Success check registered phone", resCheckPhone)
}

// GetStatus returns the connection status of a device
// Returns detailed status info for robust UX handling:
// - db_status: status stored in database (pending, active, disconnected, logged_out)
// - client_loaded: whether the WhatsApp client is in memory
// - connected: whether the client has active connection to WhatsApp servers
// - logged_in: whether the client has an authenticated session
func GetStatus(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	log.DeviceOpCtx(c, "GetStatus").Debug("Getting device status")

	// Get device info from DB first
	device, dbErr := pkgWhatsApp.GetDeviceByID(ctx, deviceID)
	dbStatus := "unknown"
	if dbErr == nil {
		dbStatus = device.Status
		log.DeviceOpCtx(c, "GetStatus").WithField("db_status", dbStatus).WithField("device_name", device.DeviceName).Debug("Device found in database")
	} else {
		log.DeviceOpCtx(c, "GetStatus").WithError(dbErr).Warn("Device not found in database")
	}

	// Check if client is loaded in memory
	clientLoaded := false
	isConnected := false
	isLoggedIn := false
	errorMessage := ""

	// Try to get client status
	err := pkgWhatsApp.WhatsAppIsClientOK(jid, deviceID)
	if err != nil {
		errorMessage = err.Error()
		// Parse the error to determine state
		if err.Error() == "WhatsApp Client is not Valid" {
			// Client not in memory - could be pending or server restarted
			clientLoaded = false
		} else if err.Error() == "WhatsApp Client is not Connected" {
			// Client loaded but not connected to WhatsApp servers
			clientLoaded = true
			isConnected = false
		} else if err.Error() == "WhatsApp Client is not Logged In" {
			// Client loaded & connected but no session
			clientLoaded = true
			isConnected = true
			isLoggedIn = false
		}
	} else {
		// All good
		clientLoaded = true
		isConnected = true
		isLoggedIn = true
	}

	log.DeviceOpCtx(c, "GetStatus").WithField("db_status", dbStatus).WithField("client_loaded", clientLoaded).WithField("is_connected", isConnected).WithField("is_logged_in", isLoggedIn).Info("Device status retrieved")

	data := map[string]interface{}{
		"db_status":     dbStatus,      // Status from database
		"client_loaded": clientLoaded,  // Is client in memory?
		"connected":     isConnected && isLoggedIn, // For backward compat: true only if fully connected
		"is_connected":  isConnected,   // Connected to WhatsApp servers
		"is_logged_in":  isLoggedIn,    // Has authenticated session
		"error":         errorMessage,
	}

	return router.ResponseSuccessWithData(c, "Device status", data)
}

// GetDevice returns details about a device
func GetDevice(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID := c.Params("device_id")

	log.Session(c, "GetDevice").WithField("target_device_id", deviceID).Info("Getting device details")

	// First try the new devices table
	device, err := pkgWhatsApp.GetDeviceByID(ctx, deviceID)
	if err == nil {
		log.Session(c, "GetDevice").WithField("target_device_id", deviceID).WithField("status", device.Status).Info("Device details retrieved")
		return router.ResponseSuccessWithData(c, "Success get device details", fiber.Map{
			"device_id":     device.DeviceID,
			"device_name":   device.DeviceName,
			"whatsmeow_jid": device.WhatsMeowJID,
			"status":        device.Status,
			"created_at":    device.CreatedAt,
			"last_active":   device.LastActiveAt,
		})
	}

	// Fallback to legacy device routing
	jid, isActive, err := pkgWhatsApp.GetWhatsMeowJID(ctx, deviceID)
	if err != nil {
		log.Session(c, "GetDevice").WithField("target_device_id", deviceID).Warn("Device not found")
		return router.ResponseNotFound(c, "Device not found")
	}

	deviceInfo := map[string]interface{}{
		"device_id":     deviceID,
		"whatsmeow_jid": jid,
		"is_active":     isActive,
	}

	log.Session(c, "GetDevice").WithField("target_device_id", deviceID).WithField("is_active", isActive).Info("Legacy device details retrieved")

	return router.ResponseSuccessWithData(c, "Success get device details", deviceInfo)
}

// Reconnect reconnects a device to WhatsApp
func Reconnect(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	log.DeviceOpCtx(c, "Reconnect").Info("Reconnecting device to WhatsApp")

	if err := pkgWhatsApp.WhatsAppReconnect(jid, deviceID); err != nil {
		log.DeviceOpCtx(c, "Reconnect").WithError(err).Error("Failed to reconnect device")
		return router.ResponseInternalError(c, err.Error())
	}

	log.DeviceOpCtx(c, "Reconnect").Info("Device reconnected successfully")

	return router.ResponseSuccess(c, "Success reconnect device")
}

// GetDeviceMe returns details about the currently authenticated device
func GetDeviceMe(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, _ := getDeviceContext(c)

	log.DeviceOpCtx(c, "GetDeviceMe").Info("Getting current device details")

	device, err := pkgWhatsApp.GetDeviceByID(ctx, deviceID)
	if err != nil {
		log.DeviceOpCtx(c, "GetDeviceMe").WithError(err).Warn("Device not found")
		return router.ResponseNotFound(c, "Device not found")
	}

	log.DeviceOpCtx(c, "GetDeviceMe").WithField("device_name", device.DeviceName).WithField("status", device.Status).Info("Current device details retrieved")

	return router.ResponseSuccessWithData(c, "Success get device details", fiber.Map{
		"device_id":     device.DeviceID,
		"device_name":   device.DeviceName,
		"whatsmeow_jid": device.WhatsMeowJID,
		"status":        device.Status,
		"created_at":    device.CreatedAt,
		"last_active":   device.LastActiveAt,
	})
}

// SetProxy sets per-device proxy configuration
func SetProxy(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	var req typWhatsApp.RequestSetProxy
	if err := c.BodyParser(&req); err != nil {
		log.DeviceOpCtx(c, "SetProxy").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed to parse body request")
	}

	log.DeviceOpCtx(c, "SetProxy").WithField("proxy_url", req.ProxyURL).Info("Setting device proxy")

	err := pkgWhatsApp.WhatsAppSetDeviceProxy(ctx, jid, deviceID, req.ProxyURL)
	if err != nil {
		log.DeviceOpCtx(c, "SetProxy").WithError(err).Error("Failed to set proxy")
		return router.ResponseInternalError(c, err.Error())
	}

	log.DeviceOpCtx(c, "SetProxy").Info("Device proxy configured successfully")

	return router.ResponseSuccess(c, "Device proxy configured successfully")
}

// GetProxy gets current device proxy configuration
func GetProxy(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, _ := getDeviceContext(c)

	log.DeviceOpCtx(c, "GetProxy").Info("Retrieving device proxy configuration")

	proxyURL, err := pkgWhatsApp.WhatsAppGetDeviceProxy(ctx, deviceID)
	if err != nil {
		log.DeviceOpCtx(c, "GetProxy").WithError(err).Error("Failed to get proxy")
		return router.ResponseInternalError(c, err.Error())
	}

	log.DeviceOpCtx(c, "GetProxy").WithField("proxy_url", proxyURL).Info("Device proxy configuration retrieved")

	return router.ResponseSuccessWithData(c, "Device proxy configuration", typWhatsApp.ResponseGetProxy{
		ProxyURL: proxyURL,
		Active:   proxyURL != "",
	})
}

// RegisterPushNotification registers device for push notifications
func RegisterPushNotification(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	var req typWhatsApp.RequestRegisterPushNotification
	if err := c.BodyParser(&req); err != nil {
		log.DeviceOpCtx(c, "RegisterPushNotification").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed to parse body request")
	}

	// Validation
	validPlatforms := map[string]bool{"fcm": true, "apns": true, "webhook": true}
	if !validPlatforms[req.Platform] {
		log.DeviceOpCtx(c, "RegisterPushNotification").Warn("Invalid platform provided")
		return router.ResponseBadRequest(c, "platform must be one of: fcm, apns, webhook")
	}

	if req.Platform != "webhook" && req.Token == "" {
		log.DeviceOpCtx(c, "RegisterPushNotification").Warn("Token required for fcm/apns platforms")
		return router.ResponseBadRequest(c, "token is required for fcm/apns platforms")
	}

	log.DeviceOpCtx(c, "RegisterPushNotification").WithField("platform", req.Platform).Info("Registering push notifications")

	err := pkgWhatsApp.WhatsAppRegisterPushNotification(ctx, jid, deviceID, req.Platform, req.Token)
	if err != nil {
		log.DeviceOpCtx(c, "RegisterPushNotification").WithError(err).Error("Failed to register push notifications")
		return router.ResponseInternalError(c, err.Error())
	}

	log.DeviceOpCtx(c, "RegisterPushNotification").Info("Push notifications registered successfully")

	return router.ResponseSuccess(c, "Push notifications registered successfully")
}

func decodeBase64Bytes(value string) ([]byte, error) {
	if value == "" {
		return nil, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err == nil {
		return decoded, nil
	}
	return base64.RawStdEncoding.DecodeString(value)
}

// GetServerPushConfig retrieves the server-side push notification config
func GetServerPushConfig(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	log.DeviceOpCtx(c, "GetServerPushConfig").Info("Retrieving server push notification config")

	config, err := pkgWhatsApp.WhatsAppGetServerPushNotificationConfig(ctx, jid, deviceID)
	if err != nil {
		log.DeviceOpCtx(c, "GetServerPushConfig").WithError(err).Error("Failed to get server push config")
		return router.ResponseInternalError(c, err.Error())
	}

	log.DeviceOpCtx(c, "GetServerPushConfig").Info("Server push config retrieved successfully")

	return router.ResponseSuccessWithData(c, "Success get server push config", config)
}

// SetServerPushConfig sets server-side push notification config
func SetServerPushConfig(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	var req typWhatsApp.RequestServerPushConfig
	if err := c.BodyParser(&req); err != nil {
		log.DeviceOpCtx(c, "SetServerPushConfig").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed to parse body request")
	}

	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	var config whatsmeow.PushConfig

	switch platform {
	case "fcm":
		if req.Token == "" {
			log.DeviceOpCtx(c, "SetServerPushConfig").Warn("Token required for fcm platform")
			return router.ResponseBadRequest(c, "token is required for fcm platform")
		}
		config = &whatsmeow.FCMPushConfig{Token: req.Token}
	case "apns":
		if req.Token == "" {
			log.DeviceOpCtx(c, "SetServerPushConfig").Warn("Token required for apns platform")
			return router.ResponseBadRequest(c, "token is required for apns platform")
		}
		keyBytes, err := decodeBase64Bytes(req.MsgIDEncKey)
		if err != nil {
			log.DeviceOpCtx(c, "SetServerPushConfig").WithError(err).Warn("Invalid msg_id_enc_key base64")
			return router.ResponseBadRequest(c, "msg_id_enc_key must be valid base64")
		}
		if len(keyBytes) > 0 && len(keyBytes) != 32 {
			log.DeviceOpCtx(c, "SetServerPushConfig").Warn("Invalid msg_id_enc_key length")
			return router.ResponseBadRequest(c, "msg_id_enc_key must be 32 bytes when provided")
		}
		config = &whatsmeow.APNsPushConfig{Token: req.Token, VoIPToken: req.VoipToken, MsgIDEncKey: keyBytes}
	case "web":
		if req.Endpoint == "" || req.Auth == "" || req.P256DH == "" {
			log.DeviceOpCtx(c, "SetServerPushConfig").Warn("Missing web push fields")
			return router.ResponseBadRequest(c, "endpoint, auth, and p256dh are required for web platform")
		}
		authBytes, err := decodeBase64Bytes(req.Auth)
		if err != nil {
			log.DeviceOpCtx(c, "SetServerPushConfig").WithError(err).Warn("Invalid auth base64")
			return router.ResponseBadRequest(c, "auth must be valid base64")
		}
		p256dhBytes, err := decodeBase64Bytes(req.P256DH)
		if err != nil {
			log.DeviceOpCtx(c, "SetServerPushConfig").WithError(err).Warn("Invalid p256dh base64")
			return router.ResponseBadRequest(c, "p256dh must be valid base64")
		}
		config = &whatsmeow.WebPushConfig{Endpoint: req.Endpoint, Auth: authBytes, P256DH: p256dhBytes}
	default:
		log.DeviceOpCtx(c, "SetServerPushConfig").Warn("Invalid platform provided")
		return router.ResponseBadRequest(c, "platform must be one of: fcm, apns, web")
	}

	log.DeviceOpCtx(c, "SetServerPushConfig").WithField("platform", platform).Info("Setting server push config")

	if err := pkgWhatsApp.WhatsAppRegisterServerPushNotifications(ctx, jid, deviceID, config); err != nil {
		log.DeviceOpCtx(c, "SetServerPushConfig").WithError(err).Error("Failed to set server push config")
		return router.ResponseInternalError(c, err.Error())
	}

	log.DeviceOpCtx(c, "SetServerPushConfig").Info("Server push config updated successfully")

	return router.ResponseSuccess(c, "Server push config updated successfully")
}

// SetForceActiveReceipts toggles active delivery receipts
func SetForceActiveReceipts(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	var req typWhatsApp.RequestForceActiveReceipts
	if err := c.BodyParser(&req); err != nil {
		log.DeviceOpCtx(c, "SetForceActiveReceipts").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed to parse body request")
	}

	log.DeviceOpCtx(c, "SetForceActiveReceipts").WithField("active", req.Active).Info("Setting force active delivery receipts")

	if err := pkgWhatsApp.WhatsAppSetForceActiveDeliveryReceipts(jid, deviceID, req.Active); err != nil {
		log.DeviceOpCtx(c, "SetForceActiveReceipts").WithError(err).Error("Failed to set force active receipts")
		return router.ResponseInternalError(c, err.Error())
	}

	log.DeviceOpCtx(c, "SetForceActiveReceipts").Info("Force active delivery receipts updated successfully")

	return router.ResponseSuccess(c, "Force active delivery receipts updated successfully")
}
