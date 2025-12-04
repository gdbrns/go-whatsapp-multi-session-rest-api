package device

import (
	"context"
	"strconv"
	"strings"

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
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	log.DeviceOp(deviceID, jid, "Login").Info("Initiating QR code login")

	var reqLogin typWhatsApp.RequestLogin
	reqLogin.Output = strings.TrimSpace(c.FormValue("output"))
	if len(reqLogin.Output) == 0 {
		reqLogin.Output = "html"
	}

	pkgWhatsApp.WhatsAppInitClient(nil, jid, deviceID)

	qrCodeImage, qrCodeTimeout, err := pkgWhatsApp.WhatsAppLogin(jid, deviceID)
	if err != nil {
		log.DeviceOp(deviceID, jid, "Login").WithError(err).Error("Failed to generate QR code")
		return router.ResponseInternalError(c, err.Error())
	}

	if qrCodeImage == "WhatsApp Client is Reconnected" {
		log.DeviceOp(deviceID, jid, "Login").Info("WhatsApp client reconnected successfully")
		return router.ResponseSuccess(c, qrCodeImage)
	}

	var resLogin typWhatsApp.ResponseLogin
	resLogin.QRCode = qrCodeImage
	resLogin.Timeout = qrCodeTimeout

	log.DeviceOp(deviceID, jid, "Login").WithField("timeout", qrCodeTimeout).WithField("output", reqLogin.Output).Info("QR code generated successfully")

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
		log.DeviceOp(deviceID, jid, "LoginWithCode").Warn("Failed to parse body request")
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	phone := strings.TrimSpace(reqLoginCode.Phone)
	if phone == "" {
		log.DeviceOp(deviceID, jid, "LoginWithCode").Warn("Phone number not provided")
		return router.ResponseBadRequest(c, "Phone is required")
	}

	log.DeviceOp(deviceID, jid, "LoginWithCode").WithField("phone", phone).Info("Initiating pairing code login")

	pkgWhatsApp.WhatsAppInitClient(nil, jid, deviceID)

	pairCode, timeout, err := pkgWhatsApp.WhatsAppLoginPair(jid, deviceID, phone)
	if err != nil {
		log.DeviceOp(deviceID, jid, "LoginWithCode").WithField("phone", phone).WithError(err).Error("Failed to generate pairing code")
		return router.ResponseInternalError(c, err.Error())
	}

	var resLoginCode typWhatsApp.ResponseLoginCode
	resLoginCode.PairCode = pairCode
	resLoginCode.Timeout = timeout

	log.DeviceOp(deviceID, jid, "LoginWithCode").WithField("phone", phone).WithField("timeout", timeout).Info("Pairing code generated successfully")

	return router.ResponseSuccessWithData(c, "Success Generate Pairing Code", resLoginCode)
}

// Logout logs out the device from WhatsApp
func Logout(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	log.DeviceOp(deviceID, jid, "Logout").Info("Logging out device")

	err := pkgWhatsApp.WhatsAppLogout(jid, deviceID)
	if err != nil {
		log.DeviceOp(deviceID, jid, "Logout").WithError(err).Error("Failed to logout device")
		return router.ResponseInternalError(c, err.Error())
	}

	log.DeviceOp(deviceID, jid, "Logout").Info("Device logged out successfully")

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

	log.DeviceOp(deviceID, jid, "CheckRegistered").WithField("phone", phone).Info("Checking if phone is registered")

	var reqCheckPhone typWhatsApp.RequestCheckPhone
	reqCheckPhone.Phone = phone

	_, err := pkgWhatsApp.WhatsAppCheckJID(ctx, jid, deviceID, reqCheckPhone.Phone)

	var resCheckPhone typWhatsApp.ResponseCheckPhone
	resCheckPhone.IsRegistered = err == nil

	log.DeviceOp(deviceID, jid, "CheckRegistered").WithField("phone", phone).WithField("is_registered", resCheckPhone.IsRegistered).Info("Phone registration check complete")

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

	log.DeviceOp(deviceID, jid, "GetStatus").Debug("Getting device status")

	// Get device info from DB first
	device, dbErr := pkgWhatsApp.GetDeviceByID(ctx, deviceID)
	dbStatus := "unknown"
	if dbErr == nil {
		dbStatus = device.Status
		log.DeviceOp(deviceID, jid, "GetStatus").WithField("db_status", dbStatus).WithField("device_name", device.DeviceName).Debug("Device found in database")
	} else {
		log.DeviceOp(deviceID, jid, "GetStatus").WithError(dbErr).Warn("Device not found in database")
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

	log.DeviceOp(deviceID, jid, "GetStatus").WithField("db_status", dbStatus).WithField("client_loaded", clientLoaded).WithField("is_connected", isConnected).WithField("is_logged_in", isLoggedIn).Info("Device status retrieved")

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

	log.DeviceOp(deviceID, jid, "Reconnect").Info("Reconnecting device to WhatsApp")

	if err := pkgWhatsApp.WhatsAppReconnect(jid, deviceID); err != nil {
		log.DeviceOp(deviceID, jid, "Reconnect").WithError(err).Error("Failed to reconnect device")
		return router.ResponseInternalError(c, err.Error())
	}

	log.DeviceOp(deviceID, jid, "Reconnect").Info("Device reconnected successfully")

	return router.ResponseSuccess(c, "Success reconnect device")
}

// GetDeviceMe returns details about the currently authenticated device
func GetDeviceMe(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, jid := getDeviceContext(c)

	log.DeviceOp(deviceID, jid, "GetDeviceMe").Info("Getting current device details")

	device, err := pkgWhatsApp.GetDeviceByID(ctx, deviceID)
	if err != nil {
		log.DeviceOp(deviceID, jid, "GetDeviceMe").WithError(err).Warn("Device not found")
		return router.ResponseNotFound(c, "Device not found")
	}

	log.DeviceOp(deviceID, jid, "GetDeviceMe").WithField("device_name", device.DeviceName).WithField("status", device.Status).Info("Current device details retrieved")

	return router.ResponseSuccessWithData(c, "Success get device details", fiber.Map{
		"device_id":     device.DeviceID,
		"device_name":   device.DeviceName,
		"whatsmeow_jid": device.WhatsMeowJID,
		"status":        device.Status,
		"created_at":    device.CreatedAt,
		"last_active":   device.LastActiveAt,
	})
}
