package admin

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"

	typAuth "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/auth/types"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/router"
	pkgWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/whatsapp"
)

// Request types for admin endpoints
type CreateAPIKeyRequest struct {
	CustomerName  string `json:"customer_name" form:"customer_name"`
	CustomerEmail string `json:"customer_email" form:"customer_email"`
	MaxDevices    int    `json:"max_devices" form:"max_devices"`
	RateLimit     int    `json:"rate_limit_per_hour" form:"rate_limit_per_hour"`
}

type UpdateAPIKeyRequest struct {
	CustomerName  string `json:"customer_name" form:"customer_name"`
	CustomerEmail string `json:"customer_email" form:"customer_email"`
	MaxDevices    int    `json:"max_devices" form:"max_devices"`
	RateLimit     int    `json:"rate_limit_per_hour" form:"rate_limit_per_hour"`
	IsActive      *bool  `json:"is_active" form:"is_active"`
}

// Helper to convert string ID to int64
func parseAPIKeyID(idStr string) (int64, error) {
	return strconv.ParseInt(idStr, 10, 64)
}

// @Summary     Create API Key
// @Description Create a new API key for a customer (Admin only)
// @Tags        Admin
// @Accept      json
// @Produce     json
// @Param       X-Admin-Secret header string true "Admin secret key"
// @Param       body body CreateAPIKeyRequest true "API Key details"
// @Success     201 {object} typAuth.ResponseAPIKeyCreated
// @Failure     400 {object} router.ResError
// @Failure     401 {object} router.ResError
// @Failure     500 {object} router.ResError
// @Router      /admin/api-keys [post]
func CreateAPIKey(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	var req CreateAPIKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return router.ResponseBadRequest(c, "Invalid request body")
	}

	if req.CustomerName == "" {
		return router.ResponseBadRequest(c, "customer_name is required")
	}

	// Set defaults
	if req.MaxDevices <= 0 {
		req.MaxDevices = 5
	}
	if req.RateLimit <= 0 {
		req.RateLimit = 1000
	}

	apiKey, err := pkgWhatsApp.CreateAPIKey(ctx, req.CustomerName, req.CustomerEmail, req.MaxDevices, req.RateLimit)
	if err != nil {
		return router.ResponseInternalError(c, "Failed to create API key: "+err.Error())
	}

	response := typAuth.ResponseAPIKeyCreated{
		ID:           int(apiKey.ID),
		APIKey:       apiKey.APIKey,
		CustomerName: apiKey.CustomerName,
		MaxDevices:   apiKey.MaxDevices,
		RateLimit:    apiKey.RateLimitPerHour,
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  true,
		"code":    fiber.StatusCreated,
		"message": "API key created successfully",
		"data":    response,
	})
}

// @Summary     List API Keys
// @Description List all API keys (Admin only)
// @Tags        Admin
// @Produce     json
// @Param       X-Admin-Secret header string true "Admin secret key"
// @Success     200 {array} pkgWhatsApp.APIKey
// @Failure     401 {object} router.ResError
// @Failure     500 {object} router.ResError
// @Router      /admin/api-keys [get]
func ListAPIKeys(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	apiKeys, err := pkgWhatsApp.ListAPIKeys(ctx)
	if err != nil {
		return router.ResponseInternalError(c, "Failed to list API keys: "+err.Error())
	}

	// Mask API keys in response (only show first/last 4 chars)
	type MaskedAPIKey struct {
		ID              int    `json:"id"`
		APIKeyMasked    string `json:"api_key_masked"`
		CustomerName    string `json:"customer_name"`
		CustomerEmail   string `json:"customer_email"`
		MaxDevices      int    `json:"max_devices"`
		RateLimitPerHour int   `json:"rate_limit_per_hour"`
		IsActive        bool   `json:"is_active"`
		DeviceCount     int    `json:"device_count"`
	}

	var masked []MaskedAPIKey
	for _, ak := range apiKeys {
		deviceCount, _ := pkgWhatsApp.CountDevicesByAPIKey(ctx, ak.ID)
		maskedKey := ak.APIKey[:8] + "..." + ak.APIKey[len(ak.APIKey)-4:]
		masked = append(masked, MaskedAPIKey{
			ID:              int(ak.ID),
			APIKeyMasked:    maskedKey,
			CustomerName:    ak.CustomerName,
			CustomerEmail:   ak.CustomerEmail,
			MaxDevices:      ak.MaxDevices,
			RateLimitPerHour: ak.RateLimitPerHour,
			IsActive:        ak.IsActive,
			DeviceCount:     deviceCount,
		})
	}

	return router.ResponseSuccessWithData(c, "API keys retrieved successfully", masked)
}

// @Summary     Get API Key
// @Description Get API key details by ID (Admin only)
// @Tags        Admin
// @Produce     json
// @Param       X-Admin-Secret header string true "Admin secret key"
// @Param       id path int true "API Key ID"
// @Success     200 {object} pkgWhatsApp.APIKey
// @Failure     401 {object} router.ResError
// @Failure     404 {object} router.ResError
// @Router      /admin/api-keys/{id} [get]
func GetAPIKey(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	idStr := c.Params("id")
	id, err := parseAPIKeyID(idStr)
	if err != nil {
		return router.ResponseBadRequest(c, "Invalid API key ID")
	}

	apiKey, err := pkgWhatsApp.GetAPIKeyByID(ctx, id)
	if err != nil {
		return router.ResponseNotFound(c, "API key not found")
	}

	// Get device count
	deviceCount, _ := pkgWhatsApp.CountDevicesByAPIKey(ctx, apiKey.ID)

	// Get devices for this API key
	devices, _ := pkgWhatsApp.ListDevicesByAPIKey(ctx, apiKey.ID)

	response := fiber.Map{
		"id":                  apiKey.ID,
		"api_key_masked":      apiKey.APIKey[:8] + "..." + apiKey.APIKey[len(apiKey.APIKey)-4:],
		"customer_name":       apiKey.CustomerName,
		"customer_email":      apiKey.CustomerEmail,
		"max_devices":         apiKey.MaxDevices,
		"rate_limit_per_hour": apiKey.RateLimitPerHour,
		"is_active":           apiKey.IsActive,
		"device_count":        deviceCount,
		"devices":             devices,
		"created_at":          apiKey.CreatedAt,
		"updated_at":          apiKey.UpdatedAt,
	}

	return router.ResponseSuccessWithData(c, "API key retrieved successfully", response)
}

// @Summary     Update API Key
// @Description Update an API key (Admin only)
// @Tags        Admin
// @Accept      json
// @Produce     json
// @Param       X-Admin-Secret header string true "Admin secret key"
// @Param       id path int true "API Key ID"
// @Param       body body UpdateAPIKeyRequest true "Update details"
// @Success     200 {object} router.ResSuccess
// @Failure     400 {object} router.ResError
// @Failure     401 {object} router.ResError
// @Failure     404 {object} router.ResError
// @Router      /admin/api-keys/{id} [patch]
func UpdateAPIKey(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	idStr := c.Params("id")
	id, err := parseAPIKeyID(idStr)
	if err != nil {
		return router.ResponseBadRequest(c, "Invalid API key ID")
	}

	// Check if API key exists
	existing, err := pkgWhatsApp.GetAPIKeyByID(ctx, id)
	if err != nil {
		return router.ResponseNotFound(c, "API key not found")
	}

	var req UpdateAPIKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return router.ResponseBadRequest(c, "Invalid request body")
	}

	// Use existing values for empty fields
	if req.CustomerName == "" {
		req.CustomerName = existing.CustomerName
	}
	if req.CustomerEmail == "" {
		req.CustomerEmail = existing.CustomerEmail
	}
	if req.MaxDevices <= 0 {
		req.MaxDevices = existing.MaxDevices
	}
	if req.RateLimit <= 0 {
		req.RateLimit = existing.RateLimitPerHour
	}

	isActive := existing.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	err = pkgWhatsApp.UpdateAPIKey(ctx, id, req.CustomerName, req.CustomerEmail, req.MaxDevices, req.RateLimit, isActive)
	if err != nil {
		return router.ResponseInternalError(c, "Failed to update API key: "+err.Error())
	}

	return router.ResponseSuccess(c, "API key updated successfully")
}

// @Summary     Delete API Key
// @Description Delete an API key and all associated devices (Admin only)
// @Tags        Admin
// @Produce     json
// @Param       X-Admin-Secret header string true "Admin secret key"
// @Param       id path int true "API Key ID"
// @Success     200 {object} router.ResSuccess
// @Failure     401 {object} router.ResError
// @Failure     404 {object} router.ResError
// @Router      /admin/api-keys/{id} [delete]
func DeleteAPIKey(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	idStr := c.Params("id")
	id, err := parseAPIKeyID(idStr)
	if err != nil {
		return router.ResponseBadRequest(c, "Invalid API key ID")
	}

	// Check if API key exists
	_, err = pkgWhatsApp.GetAPIKeyByID(ctx, id)
	if err != nil {
		return router.ResponseNotFound(c, "API key not found")
	}

	err = pkgWhatsApp.DeleteAPIKey(ctx, id)
	if err != nil {
		return router.ResponseInternalError(c, "Failed to delete API key: "+err.Error())
	}

	return router.ResponseSuccess(c, "API key deleted successfully")
}

// @Summary     List Devices by API Key
// @Description List all devices for a specific API key (Admin only)
// @Tags        Admin
// @Produce     json
// @Param       X-Admin-Secret header string true "Admin secret key"
// @Param       id path int true "API Key ID"
// @Success     200 {array} pkgWhatsApp.Device
// @Failure     401 {object} router.ResError
// @Failure     404 {object} router.ResError
// @Router      /admin/api-keys/{id}/devices [get]
func ListDevicesByAPIKey(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	idStr := c.Params("id")
	id, err := parseAPIKeyID(idStr)
	if err != nil {
		return router.ResponseBadRequest(c, "Invalid API key ID")
	}

	// Check if API key exists
	_, err = pkgWhatsApp.GetAPIKeyByID(ctx, id)
	if err != nil {
		return router.ResponseNotFound(c, "API key not found")
	}

	devices, err := pkgWhatsApp.ListDevicesByAPIKey(ctx, id)
	if err != nil {
		return router.ResponseInternalError(c, "Failed to list devices: "+err.Error())
	}

	// Mask device secrets
	type MaskedDevice struct {
		DeviceID     string  `json:"device_id"`
		DeviceName   string  `json:"device_name"`
		WhatsmeowJID string  `json:"whatsmeow_jid"`
		Status       string  `json:"status"`
		CreatedAt    string  `json:"created_at"`
		LastActiveAt *string `json:"last_active_at"`
	}

	var masked []MaskedDevice
	for _, d := range devices {
		var lastActive *string
		if d.LastActiveAt != nil {
			ts := d.LastActiveAt.Format("2006-01-02T15:04:05Z07:00")
			lastActive = &ts
		}
		masked = append(masked, MaskedDevice{
			DeviceID:     d.DeviceID,
			DeviceName:   d.DeviceName,
			WhatsmeowJID: d.WhatsMeowJID,
			Status:       d.Status,
			CreatedAt:    d.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			LastActiveAt: lastActive,
		})
	}

	return router.ResponseSuccessWithData(c, "Devices retrieved successfully", masked)
}

// @Summary     Delete Device
// @Description Delete a device by ID (Admin only)
// @Tags        Admin
// @Produce     json
// @Param       X-Admin-Secret header string true "Admin secret key"
// @Param       device_id path string true "Device ID (UUID)"
// @Success     200 {object} router.ResSuccess
// @Failure     401 {object} router.ResError
// @Failure     404 {object} router.ResError
// @Router      /admin/devices/{device_id} [delete]
func DeleteDevice(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	deviceID := c.Params("device_id")

	// Check if device exists
	_, err := pkgWhatsApp.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return router.ResponseNotFound(c, "Device not found")
	}

	err = pkgWhatsApp.DeleteDevice(ctx, deviceID)
	if err != nil {
		return router.ResponseInternalError(c, "Failed to delete device: "+err.Error())
	}

	return router.ResponseSuccess(c, "Device deleted successfully")
}
