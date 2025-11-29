package auth

import (
	"context"

	"github.com/gofiber/fiber/v2"

	typAuth "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/auth/types"
	pkgAuth "github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/auth"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/router"
	pkgWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/whatsapp"
)

// CreateDevice creates a new device using API Key authentication
// @Summary     Create Device
// @Description Create a new device for the API key. Returns device_id, device_secret, and JWT token.
// @Tags        Device Management
// @Accept      json
// @Produce     json
// @Param       X-API-Key header string true "API Key"
// @Param       body body typAuth.RequestCreateDevice false "Device details (optional)"
// @Success     201 {object} typAuth.ResponseDeviceCreated
// @Failure     400 {object} router.ResError
// @Failure     401 {object} router.ResError
// @Failure     500 {object} router.ResError
// @Router      /devices [post]
func CreateDevice(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	// Get API key from context (set by APIKeyAuth middleware)
	apiKey := c.Locals("api_key").(*pkgWhatsApp.APIKey)

	// Parse request body
	var req typAuth.RequestCreateDevice
	_ = c.BodyParser(&req)

	// Check device limit
	deviceCount, err := pkgWhatsApp.CountDevicesByAPIKey(ctx, apiKey.ID)
	if err != nil {
		return router.ResponseInternalError(c, "Failed to check device count")
	}

	if deviceCount >= apiKey.MaxDevices {
		return router.ResponseBadRequest(c, "Device limit reached")
	}

	// Create device
	device, err := pkgWhatsApp.CreateDevice(ctx, apiKey.ID, req.DeviceName)
	if err != nil {
		return router.ResponseInternalError(c, "Failed to create device: "+err.Error())
	}

	// Also create device routing for whatsmeow compatibility
	err = pkgWhatsApp.SaveDeviceRouting(ctx, device.DeviceID, "")
	if err != nil {
		// Rollback device creation
		_ = pkgWhatsApp.DeleteDevice(ctx, device.DeviceID)
		return router.ResponseInternalError(c, "Failed to create device routing")
	}

	// Generate JWT token for the device
	token, err := pkgAuth.GenerateDeviceToken(device.DeviceID, device.APIKeyID, "", device.JWTVersion)
	if err != nil {
		return router.ResponseInternalError(c, "Failed to generate device token: "+err.Error())
	}

	response := typAuth.ResponseDeviceCreated{
		DeviceID:     device.DeviceID,
		DeviceSecret: device.DeviceSecret,
		DeviceName:   device.DeviceName,
		Token:        token,
		Message:      "Device created successfully. Save the device_secret securely - it's needed to regenerate tokens. Use the token in Authorization header for all API calls.",
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  true,
		"code":    fiber.StatusCreated,
		"message": "Device created successfully",
		"data":    response,
	})
}

// RegenerateToken regenerates a JWT token for a device using device credentials
// @Summary     Regenerate Device Token
// @Description Regenerate a new JWT token using device_id and device_secret. Invalidates all previous tokens.
// @Tags        Device Management
// @Accept      json
// @Produce     json
// @Param       body body typAuth.RequestRegenerateToken true "Device credentials"
// @Success     200 {object} typAuth.ResponseTokenRegenerated
// @Failure     400 {object} router.ResError
// @Failure     401 {object} router.ResError
// @Failure     500 {object} router.ResError
// @Router      /devices/token [post]
func RegenerateToken(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	var req typAuth.RequestRegenerateToken
	if err := c.BodyParser(&req); err != nil {
		return router.ResponseBadRequest(c, "Invalid request body")
	}

	if req.DeviceID == "" || req.DeviceSecret == "" {
		return router.ResponseBadRequest(c, "device_id and device_secret are required")
	}

	// Validate device credentials
	device, err := pkgWhatsApp.ValidateDeviceCredentials(ctx, req.DeviceID, req.DeviceSecret)
	if err != nil {
		return router.ResponseUnauthorized(c, "Invalid device credentials")
	}

	// Increment JWT version to invalidate old tokens
	newVersion, err := pkgWhatsApp.IncrementDeviceJWTVersion(ctx, device.DeviceID)
	if err != nil {
		return router.ResponseInternalError(c, "Failed to invalidate old tokens")
	}

	// Generate new JWT token with incremented version
	token, err := pkgAuth.GenerateDeviceToken(device.DeviceID, device.APIKeyID, device.WhatsMeowJID, newVersion)
	if err != nil {
		return router.ResponseInternalError(c, "Failed to generate new token: "+err.Error())
	}

	response := typAuth.ResponseTokenRegenerated{
		DeviceID: device.DeviceID,
		Token:    token,
		Message:  "Token regenerated successfully. All previous tokens are now invalid.",
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  true,
		"code":    fiber.StatusOK,
		"message": "Token regenerated successfully",
		"data":    response,
	})
}
