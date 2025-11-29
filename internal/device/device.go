package device

import (
	"context"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	typWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/types"
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

	routings, err := pkgWhatsApp.ListDeviceRoutings(ctx)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

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

	var reqLogin typWhatsApp.RequestLogin
	reqLogin.Output = strings.TrimSpace(c.FormValue("output"))
	if len(reqLogin.Output) == 0 {
		reqLogin.Output = "html"
	}

	pkgWhatsApp.WhatsAppInitClient(nil, jid, deviceID)

	qrCodeImage, qrCodeTimeout, err := pkgWhatsApp.WhatsAppLogin(jid, deviceID)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	if qrCodeImage == "WhatsApp Client is Reconnected" {
		return router.ResponseSuccess(c, qrCodeImage)
	}

	var resLogin typWhatsApp.ResponseLogin
	resLogin.QRCode = qrCodeImage
	resLogin.Timeout = qrCodeTimeout

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
		return router.ResponseBadRequest(c, "Failed parse body request")
	}

	phone := strings.TrimSpace(reqLoginCode.Phone)
	if phone == "" {
		return router.ResponseBadRequest(c, "Phone is required")
	}

	pkgWhatsApp.WhatsAppInitClient(nil, jid, deviceID)

	pairCode, timeout, err := pkgWhatsApp.WhatsAppLoginPair(jid, deviceID, phone)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	var resLoginCode typWhatsApp.ResponseLoginCode
	resLoginCode.PairCode = pairCode
	resLoginCode.Timeout = timeout

	return router.ResponseSuccessWithData(c, "Success Generate Pairing Code", resLoginCode)
}

// Logout logs out the device from WhatsApp
func Logout(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	err := pkgWhatsApp.WhatsAppLogout(jid, deviceID)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

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

	var reqCheckPhone typWhatsApp.RequestCheckPhone
	reqCheckPhone.Phone = phone

	_, err := pkgWhatsApp.WhatsAppCheckJID(ctx, jid, deviceID, reqCheckPhone.Phone)

	var resCheckPhone typWhatsApp.ResponseCheckPhone
	resCheckPhone.IsRegistered = err == nil

	return router.ResponseSuccessWithData(c, "Success check registered phone", resCheckPhone)
}

// GetStatus returns the connection status of a device
func GetStatus(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	isConnected := pkgWhatsApp.WhatsAppIsClientOK(jid, deviceID)

	data := map[string]interface{}{
		"connected": isConnected,
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

	// First try the new devices table
	device, err := pkgWhatsApp.GetDeviceByID(ctx, deviceID)
	if err == nil {
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
		return router.ResponseNotFound(c, "Device not found")
	}

	deviceInfo := map[string]interface{}{
		"device_id":     deviceID,
		"whatsmeow_jid": jid,
		"is_active":     isActive,
	}

	return router.ResponseSuccessWithData(c, "Success get device details", deviceInfo)
}

// Reconnect reconnects a device to WhatsApp
func Reconnect(c *fiber.Ctx) error {
	deviceID, jid := getDeviceContext(c)

	if err := pkgWhatsApp.WhatsAppReconnect(jid, deviceID); err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccess(c, "Success reconnect device")
}

// GetDeviceMe returns details about the currently authenticated device
func GetDeviceMe(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID, _ := getDeviceContext(c)

	device, err := pkgWhatsApp.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return router.ResponseNotFound(c, "Device not found")
	}

	return router.ResponseSuccessWithData(c, "Success get device details", fiber.Map{
		"device_id":     device.DeviceID,
		"device_name":   device.DeviceName,
		"whatsmeow_jid": device.WhatsMeowJID,
		"status":        device.Status,
		"created_at":    device.CreatedAt,
		"last_active":   device.LastActiveAt,
	})
}
