package admin

import (
	"context"
	"runtime"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	typAuth "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/auth/types"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/log"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/router"
	pkgWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/whatsapp"
)

var startTime = time.Now()

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

	log.AdminOp(c, "CreateAPIKey").Info("Creating new API key")

	var req CreateAPIKeyRequest
	if err := c.BodyParser(&req); err != nil {
		log.AdminOp(c, "CreateAPIKey").Warn("Invalid request body")
		return router.ResponseBadRequest(c, "Invalid request body")
	}

	if req.CustomerName == "" {
		log.AdminOp(c, "CreateAPIKey").Warn("Missing customer_name")
		return router.ResponseBadRequest(c, "customer_name is required")
	}

	// Set defaults
	if req.MaxDevices <= 0 {
		req.MaxDevices = 5
	}
	if req.RateLimit <= 0 {
		req.RateLimit = 1000
	}

	log.AdminOp(c, "CreateAPIKey").WithField("customer_name", req.CustomerName).WithField("max_devices", req.MaxDevices).Info("Creating API key for customer")

	apiKey, err := pkgWhatsApp.CreateAPIKey(ctx, req.CustomerName, req.CustomerEmail, req.MaxDevices, req.RateLimit)
	if err != nil {
		log.AdminOp(c, "CreateAPIKey").WithError(err).Error("Failed to create API key")
		return router.ResponseInternalError(c, "Failed to create API key: "+err.Error())
	}

	response := typAuth.ResponseAPIKeyCreated{
		ID:           int(apiKey.ID),
		APIKey:       apiKey.APIKey,
		CustomerName: apiKey.CustomerName,
		MaxDevices:   apiKey.MaxDevices,
		RateLimit:    apiKey.RateLimitPerHour,
	}

	log.AdminOp(c, "CreateAPIKey").WithField("api_key_id", apiKey.ID).WithField("customer_name", req.CustomerName).Info("API key created successfully")

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

	log.AdminOp(c, "ListAPIKeys").Info("Listing all API keys")

	apiKeys, err := pkgWhatsApp.ListAPIKeys(ctx)
	if err != nil {
		log.AdminOp(c, "ListAPIKeys").WithError(err).Error("Failed to list API keys")
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

	log.AdminOp(c, "ListAPIKeys").WithField("count", len(masked)).Info("API keys listed successfully")

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
		log.AdminOp(c, "GetAPIKey").WithField("id_str", idStr).Warn("Invalid API key ID")
		return router.ResponseBadRequest(c, "Invalid API key ID")
	}

	log.AdminOp(c, "GetAPIKey").WithField("api_key_id", id).Info("Getting API key details")

	apiKey, err := pkgWhatsApp.GetAPIKeyByID(ctx, id)
	if err != nil {
		log.AdminOp(c, "GetAPIKey").WithField("api_key_id", id).Warn("API key not found")
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

	log.AdminOp(c, "GetAPIKey").WithField("api_key_id", id).WithField("device_count", deviceCount).Info("API key retrieved successfully")

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
		log.AdminOp(c, "UpdateAPIKey").WithField("id_str", idStr).Warn("Invalid API key ID")
		return router.ResponseBadRequest(c, "Invalid API key ID")
	}

	log.AdminOp(c, "UpdateAPIKey").WithField("api_key_id", id).Info("Updating API key")

	// Check if API key exists
	existing, err := pkgWhatsApp.GetAPIKeyByID(ctx, id)
	if err != nil {
		log.AdminOp(c, "UpdateAPIKey").WithField("api_key_id", id).Warn("API key not found")
		return router.ResponseNotFound(c, "API key not found")
	}

	var req UpdateAPIKeyRequest
	if err := c.BodyParser(&req); err != nil {
		log.AdminOp(c, "UpdateAPIKey").WithField("api_key_id", id).Warn("Invalid request body")
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
		log.AdminOp(c, "UpdateAPIKey").WithField("api_key_id", id).WithError(err).Error("Failed to update API key")
		return router.ResponseInternalError(c, "Failed to update API key: "+err.Error())
	}

	log.AdminOp(c, "UpdateAPIKey").WithField("api_key_id", id).WithField("is_active", isActive).Info("API key updated successfully")

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
		log.AdminOp(c, "DeleteAPIKey").WithField("id_str", idStr).Warn("Invalid API key ID")
		return router.ResponseBadRequest(c, "Invalid API key ID")
	}

	log.AdminOp(c, "DeleteAPIKey").WithField("api_key_id", id).Info("Deleting API key")

	// Check if API key exists
	_, err = pkgWhatsApp.GetAPIKeyByID(ctx, id)
	if err != nil {
		log.AdminOp(c, "DeleteAPIKey").WithField("api_key_id", id).Warn("API key not found")
		return router.ResponseNotFound(c, "API key not found")
	}

	err = pkgWhatsApp.DeleteAPIKey(ctx, id)
	if err != nil {
		log.AdminOp(c, "DeleteAPIKey").WithField("api_key_id", id).WithError(err).Error("Failed to delete API key")
		return router.ResponseInternalError(c, "Failed to delete API key: "+err.Error())
	}

	log.AdminOp(c, "DeleteAPIKey").WithField("api_key_id", id).Info("API key deleted successfully")

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
		log.AdminOp(c, "ListDevicesByAPIKey").WithField("id_str", idStr).Warn("Invalid API key ID")
		return router.ResponseBadRequest(c, "Invalid API key ID")
	}

	log.AdminOp(c, "ListDevicesByAPIKey").WithField("api_key_id", id).Info("Listing devices for API key")

	// Check if API key exists
	_, err = pkgWhatsApp.GetAPIKeyByID(ctx, id)
	if err != nil {
		log.AdminOp(c, "ListDevicesByAPIKey").WithField("api_key_id", id).Warn("API key not found")
		return router.ResponseNotFound(c, "API key not found")
	}

	devices, err := pkgWhatsApp.ListDevicesByAPIKey(ctx, id)
	if err != nil {
		log.AdminOp(c, "ListDevicesByAPIKey").WithField("api_key_id", id).WithError(err).Error("Failed to list devices")
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

	log.AdminOp(c, "ListDevicesByAPIKey").WithField("api_key_id", id).WithField("device_count", len(masked)).Info("Devices listed successfully")

	return router.ResponseSuccessWithData(c, "Devices retrieved successfully", masked)
}

// @Summary     Get All Device Statuses by API Key
// @Description Get live connection status for all devices of an API key (Admin only)
// @Tags        Admin
// @Produce     json
// @Param       X-Admin-Secret header string true "Admin secret key"
// @Param       id path int true "API Key ID"
// @Success     200 {object} router.ResSuccess
// @Failure     401 {object} router.ResError
// @Failure     404 {object} router.ResError
// @Router      /admin/api-keys/{id}/devices/status [get]
func GetAllDeviceStatuses(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	idStr := c.Params("id")
	id, err := parseAPIKeyID(idStr)
	if err != nil {
		log.AdminOp(c, "GetAllDeviceStatuses").WithField("id_str", idStr).Warn("Invalid API key ID")
		return router.ResponseBadRequest(c, "Invalid API key ID")
	}

	log.AdminOp(c, "GetAllDeviceStatuses").WithField("api_key_id", id).Info("Getting device statuses for API key")

	// Check if API key exists
	_, err = pkgWhatsApp.GetAPIKeyByID(ctx, id)
	if err != nil {
		log.AdminOp(c, "GetAllDeviceStatuses").WithField("api_key_id", id).Warn("API key not found")
		return router.ResponseNotFound(c, "API key not found")
	}

	devices, err := pkgWhatsApp.ListDevicesByAPIKey(ctx, id)
	if err != nil {
		log.AdminOp(c, "GetAllDeviceStatuses").WithField("api_key_id", id).WithError(err).Error("Failed to list devices")
		return router.ResponseInternalError(c, "Failed to list devices: "+err.Error())
	}

	// Build response with live status for each device
	type DeviceStatus struct {
		DeviceID     string  `json:"device_id"`
		DeviceName   string  `json:"device_name"`
		WhatsmeowJID string  `json:"whatsmeow_jid"`
		DBStatus     string  `json:"db_status"`
		ClientLoaded bool    `json:"client_loaded"`
		Connected    bool    `json:"connected"`
		IsConnected  bool    `json:"is_connected"`
		IsLoggedIn   bool    `json:"is_logged_in"`
		Error        string  `json:"error,omitempty"`
	}

	var statuses []DeviceStatus
	connectedCount := 0
	for _, d := range devices {
		status := DeviceStatus{
			DeviceID:     d.DeviceID,
			DeviceName:   d.DeviceName,
			WhatsmeowJID: d.WhatsMeowJID,
			DBStatus:     d.Status,
		}

		// Check live status using the device's JID
		jid := ""
		if d.WhatsMeowJID != "" {
			jid = pkgWhatsApp.WhatsAppDecomposeJID(d.WhatsMeowJID)
		}

		err := pkgWhatsApp.WhatsAppIsClientOK(jid, d.DeviceID)
		if err != nil {
			status.Error = err.Error()
			// Parse the error to determine state
			if err.Error() == "WhatsApp Client is not Valid" {
				status.ClientLoaded = false
			} else if err.Error() == "WhatsApp Client is not Connected" {
				status.ClientLoaded = true
				status.IsConnected = false
			} else if err.Error() == "WhatsApp Client is not Logged In" {
				status.ClientLoaded = true
				status.IsConnected = true
				status.IsLoggedIn = false
			}
		} else {
			status.ClientLoaded = true
			status.IsConnected = true
			status.IsLoggedIn = true
			status.Connected = true
			connectedCount++
		}

		statuses = append(statuses, status)
	}

	log.AdminOp(c, "GetAllDeviceStatuses").WithField("api_key_id", id).WithField("total_devices", len(statuses)).WithField("connected_devices", connectedCount).Info("Device statuses retrieved successfully")

	return router.ResponseSuccessWithData(c, "Device statuses retrieved successfully", statuses)
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

	log.AdminOp(c, "DeleteDevice").WithField("device_id", deviceID).Info("Deleting device")

	// Check if device exists
	device, err := pkgWhatsApp.GetDeviceByID(ctx, deviceID)
	if err != nil {
		log.AdminOp(c, "DeleteDevice").WithField("device_id", deviceID).Warn("Device not found")
		return router.ResponseNotFound(c, "Device not found")
	}

	err = pkgWhatsApp.DeleteDevice(ctx, deviceID)
	if err != nil {
		log.AdminOp(c, "DeleteDevice").WithField("device_id", deviceID).WithError(err).Error("Failed to delete device")
		return router.ResponseInternalError(c, "Failed to delete device: "+err.Error())
	}

	log.AdminOp(c, "DeleteDevice").WithField("device_id", deviceID).WithField("device_name", device.DeviceName).Info("Device deleted successfully")

	return router.ResponseSuccess(c, "Device deleted successfully")
}

// @Summary     Get Admin Stats
// @Description Get system-wide statistics for admin dashboard (Admin only)
// @Tags        Admin
// @Produce     json
// @Param       X-Admin-Secret header string true "Admin secret key"
// @Success     200 {object} router.ResSuccess
// @Failure     401 {object} router.ResError
// @Failure     500 {object} router.ResError
// @Router      /admin/stats [get]
func GetStats(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	log.AdminOp(c, "GetStats").Info("Getting admin statistics")

	stats, err := pkgWhatsApp.GetAdminStats(ctx)
	if err != nil {
		log.AdminOp(c, "GetStats").WithError(err).Error("Failed to get admin stats")
		return router.ResponseInternalError(c, "Failed to get statistics: "+err.Error())
	}

	log.AdminOp(c, "GetStats").WithField("total_devices", stats.TotalDevices).WithField("connected_devices", stats.ConnectedDevices).Info("Admin stats retrieved successfully")

	return router.ResponseSuccessWithData(c, "Statistics retrieved successfully", stats)
}

// @Summary     List All Devices
// @Description Get all devices across all API keys with customer info (Admin only)
// @Tags        Admin
// @Produce     json
// @Param       X-Admin-Secret header string true "Admin secret key"
// @Success     200 {object} router.ResSuccess
// @Failure     401 {object} router.ResError
// @Failure     500 {object} router.ResError
// @Router      /admin/devices [get]
func ListAllDevices(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	log.AdminOp(c, "ListAllDevices").Info("Listing all devices")

	devices, err := pkgWhatsApp.ListAllDevices(ctx)
	if err != nil {
		log.AdminOp(c, "ListAllDevices").WithError(err).Error("Failed to list all devices")
		return router.ResponseInternalError(c, "Failed to list devices: "+err.Error())
	}

	// Build response with masked data
	type DeviceResponse struct {
		DeviceID     string  `json:"device_id"`
		DeviceName   string  `json:"device_name"`
		APIKeyID     int64   `json:"api_key_id"`
		CustomerName string  `json:"customer_name"`
		WhatsmeowJID string  `json:"whatsmeow_jid"`
		Status       string  `json:"status"`
		CreatedAt    string  `json:"created_at"`
		LastActiveAt *string `json:"last_active_at"`
	}

	var response []DeviceResponse
	connectedCount := 0
	disconnectedCount := 0
	pendingCount := 0

	for _, d := range devices {
		var lastActive *string
		if d.LastActiveAt != nil {
			ts := d.LastActiveAt.Format("2006-01-02T15:04:05Z07:00")
			lastActive = &ts
		}

		response = append(response, DeviceResponse{
			DeviceID:     d.DeviceID,
			DeviceName:   d.DeviceName,
			APIKeyID:     d.APIKeyID,
			CustomerName: d.CustomerName,
			WhatsmeowJID: d.WhatsMeowJID,
			Status:       d.Status,
			CreatedAt:    d.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			LastActiveAt: lastActive,
		})

		switch d.Status {
		case "active":
			connectedCount++
		case "disconnected", "logged_out":
			disconnectedCount++
		case "pending":
			pendingCount++
		}
	}

	summary := fiber.Map{
		"total":        len(devices),
		"connected":    connectedCount,
		"disconnected": disconnectedCount,
		"pending":      pendingCount,
	}

	log.AdminOp(c, "ListAllDevices").WithField("total_devices", len(devices)).Info("All devices listed successfully")

	return router.ResponseSuccessWithData(c, "Devices retrieved successfully", fiber.Map{
		"devices": response,
		"summary": summary,
	})
}

// @Summary     Get All Device Statuses
// @Description Get live connection status for all devices across all API keys (Admin only)
// @Tags        Admin
// @Produce     json
// @Param       X-Admin-Secret header string true "Admin secret key"
// @Success     200 {object} router.ResSuccess
// @Failure     401 {object} router.ResError
// @Failure     500 {object} router.ResError
// @Router      /admin/devices/status [get]
func GetAllDevicesStatus(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	log.AdminOp(c, "GetAllDevicesStatus").Info("Getting all device statuses")

	devices, err := pkgWhatsApp.ListAllDevices(ctx)
	if err != nil {
		log.AdminOp(c, "GetAllDevicesStatus").WithError(err).Error("Failed to list devices")
		return router.ResponseInternalError(c, "Failed to list devices: "+err.Error())
	}

	// Build response with live status for each device
	type DeviceStatusResponse struct {
		DeviceID     string `json:"device_id"`
		DeviceName   string `json:"device_name"`
		CustomerName string `json:"customer_name"`
		WhatsmeowJID string `json:"whatsmeow_jid"`
		DBStatus     string `json:"db_status"`
		ClientLoaded bool   `json:"client_loaded"`
		Connected    bool   `json:"connected"`
		IsConnected  bool   `json:"is_connected"`
		IsLoggedIn   bool   `json:"is_logged_in"`
		Error        string `json:"error,omitempty"`
	}

	var statuses []DeviceStatusResponse
	onlineCount := 0
	offlineCount := 0

	for _, d := range devices {
		status := DeviceStatusResponse{
			DeviceID:     d.DeviceID,
			DeviceName:   d.DeviceName,
			CustomerName: d.CustomerName,
			WhatsmeowJID: d.WhatsMeowJID,
			DBStatus:     d.Status,
		}

		// Check live status using the device's JID
		jid := ""
		if d.WhatsMeowJID != "" {
			jid = pkgWhatsApp.WhatsAppDecomposeJID(d.WhatsMeowJID)
		}

		err := pkgWhatsApp.WhatsAppIsClientOK(jid, d.DeviceID)
		if err != nil {
			status.Error = err.Error()
			if err.Error() == "WhatsApp Client is not Valid" {
				status.ClientLoaded = false
			} else if err.Error() == "WhatsApp Client is not Connected" {
				status.ClientLoaded = true
				status.IsConnected = false
			} else if err.Error() == "WhatsApp Client is not Logged In" {
				status.ClientLoaded = true
				status.IsConnected = true
				status.IsLoggedIn = false
			}
			offlineCount++
		} else {
			status.ClientLoaded = true
			status.IsConnected = true
			status.IsLoggedIn = true
			status.Connected = true
			onlineCount++
		}

		statuses = append(statuses, status)
	}

	summary := fiber.Map{
		"total":   len(devices),
		"online":  onlineCount,
		"offline": offlineCount,
	}

	log.AdminOp(c, "GetAllDevicesStatus").WithField("total", len(devices)).WithField("online", onlineCount).WithField("offline", offlineCount).Info("All device statuses retrieved successfully")

	return router.ResponseSuccessWithData(c, "Device statuses retrieved successfully", fiber.Map{
		"devices": statuses,
		"summary": summary,
	})
}

// @Summary     Get System Health
// @Description Get system health check information (Admin only)
// @Tags        Admin
// @Produce     json
// @Param       X-Admin-Secret header string true "Admin secret key"
// @Success     200 {object} router.ResSuccess
// @Failure     401 {object} router.ResError
// @Failure     500 {object} router.ResError
// @Router      /admin/health [get]
func GetHealth(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	log.AdminOp(c, "GetHealth").Info("Getting system health")

	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Get uptime
	uptime := time.Since(startTime)
	uptimeStr := formatDuration(uptime)

	// Get stats
	stats, _ := pkgWhatsApp.GetAdminStats(ctx)

	// Check database connectivity
	dbStatus := "ok"
	devices, err := pkgWhatsApp.ListAllDevices(ctx)
	if err != nil {
		dbStatus = "error: " + err.Error()
	}

	// Count connected WhatsApp clients
	connectedClients := 0
	for _, d := range devices {
		jid := ""
		if d.WhatsMeowJID != "" {
			jid = pkgWhatsApp.WhatsAppDecomposeJID(d.WhatsMeowJID)
		}
		if pkgWhatsApp.WhatsAppIsClientOK(jid, d.DeviceID) == nil {
			connectedClients++
		}
	}

	health := fiber.Map{
		"status":            "ok",
		"database":          dbStatus,
		"whatsapp_clients":  connectedClients,
		"total_devices":     len(devices),
		"memory_alloc":      formatBytes(memStats.Alloc),
		"memory_sys":        formatBytes(memStats.Sys),
		"goroutines":        runtime.NumGoroutine(),
		"uptime":            uptimeStr,
		"uptime_seconds":    int64(uptime.Seconds()),
		"version":           "2.0.0",
		"go_version":        runtime.Version(),
	}

	if stats != nil {
		health["api_keys"] = stats.TotalAPIKeys
		health["active_api_keys"] = stats.ActiveAPIKeys
	}

	log.AdminOp(c, "GetHealth").WithField("status", "ok").WithField("connected_clients", connectedClients).Info("Health check completed")

	return router.ResponseSuccessWithData(c, "System health retrieved successfully", health)
}

// @Summary     Get Webhook Stats
// @Description Get webhook delivery statistics (Admin only)
// @Tags        Admin
// @Produce     json
// @Param       X-Admin-Secret header string true "Admin secret key"
// @Success     200 {object} router.ResSuccess
// @Failure     401 {object} router.ResError
// @Failure     500 {object} router.ResError
// @Router      /admin/webhooks/stats [get]
func GetWebhookStats(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	log.AdminOp(c, "GetWebhookStats").Info("Getting webhook statistics")

	stats, err := pkgWhatsApp.GetWebhookStats(ctx)
	if err != nil {
		log.AdminOp(c, "GetWebhookStats").WithError(err).Error("Failed to get webhook stats")
		return router.ResponseInternalError(c, "Failed to get webhook statistics: "+err.Error())
	}

	log.AdminOp(c, "GetWebhookStats").WithField("total_webhooks", stats["total_webhooks"]).Info("Webhook stats retrieved successfully")

	return router.ResponseSuccessWithData(c, "Webhook statistics retrieved successfully", stats)
}

// @Summary     Reconnect All Devices
// @Description Attempt to reconnect all disconnected devices (Admin only)
// @Tags        Admin
// @Produce     json
// @Param       X-Admin-Secret header string true "Admin secret key"
// @Success     200 {object} router.ResSuccess
// @Failure     401 {object} router.ResError
// @Failure     500 {object} router.ResError
// @Router      /admin/devices/reconnect [post]
func ReconnectAllDevices(c *fiber.Ctx) error {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	log.AdminOp(c, "ReconnectAllDevices").Info("Attempting to reconnect all devices")

	devices, err := pkgWhatsApp.ListAllDevices(ctx)
	if err != nil {
		log.AdminOp(c, "ReconnectAllDevices").WithError(err).Error("Failed to list devices")
		return router.ResponseInternalError(c, "Failed to list devices: "+err.Error())
	}

	type ReconnectResult struct {
		DeviceID   string `json:"device_id"`
		DeviceName string `json:"device_name"`
		Status     string `json:"status"`
		Error      string `json:"error,omitempty"`
	}

	var results []ReconnectResult
	successCount := 0
	failedCount := 0
	skippedCount := 0

	for _, d := range devices {
		result := ReconnectResult{
			DeviceID:   d.DeviceID,
			DeviceName: d.DeviceName,
		}

		// Check if device has a JID (was logged in before)
		if d.WhatsMeowJID == "" {
			result.Status = "skipped"
			result.Error = "Device never logged in"
			skippedCount++
			results = append(results, result)
			continue
		}

		jid := pkgWhatsApp.WhatsAppDecomposeJID(d.WhatsMeowJID)

		// Check if already connected
		if pkgWhatsApp.WhatsAppIsClientOK(jid, d.DeviceID) == nil {
			result.Status = "already_connected"
			skippedCount++
			results = append(results, result)
			continue
		}

		// Try to reconnect
		err := pkgWhatsApp.WhatsAppReconnect(jid, d.DeviceID)
		if err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			failedCount++
		} else {
			result.Status = "reconnected"
			successCount++
		}

		results = append(results, result)
	}

	summary := fiber.Map{
		"total":       len(devices),
		"reconnected": successCount,
		"failed":      failedCount,
		"skipped":     skippedCount,
	}

	log.AdminOp(c, "ReconnectAllDevices").WithField("reconnected", successCount).WithField("failed", failedCount).WithField("skipped", skippedCount).Info("Reconnect operation completed")

	return router.ResponseSuccessWithData(c, "Reconnect operation completed", fiber.Map{
		"results": results,
		"summary": summary,
	})
}

// Helper function to format duration
func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return strconv.Itoa(days) + "d " + strconv.Itoa(hours) + "h " + strconv.Itoa(minutes) + "m " + strconv.Itoa(seconds) + "s"
	}
	if hours > 0 {
		return strconv.Itoa(hours) + "h " + strconv.Itoa(minutes) + "m " + strconv.Itoa(seconds) + "s"
	}
	if minutes > 0 {
		return strconv.Itoa(minutes) + "m " + strconv.Itoa(seconds) + "s"
	}
	return strconv.Itoa(seconds) + "s"
}

// Helper function to format bytes
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return strconv.FormatUint(bytes, 10) + " B"
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return strconv.FormatFloat(float64(bytes)/float64(div), 'f', 1, 64) + " " + []string{"KB", "MB", "GB", "TB"}[exp]
}
