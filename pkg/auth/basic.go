package auth

import (
	"context"
	"crypto/subtle"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/router"
	pkgWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/whatsapp"
)

// AdminAuth validates the X-Admin-Secret header for admin endpoints
func AdminAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		adminSecret := c.Get("X-Admin-Secret")
		if adminSecret == "" {
			return router.ResponseUnauthorized(c, "Missing X-Admin-Secret header")
		}

		if AdminSecretKey == "" {
			return router.ResponseInternalError(c, "Admin secret key not configured")
		}

		if subtle.ConstantTimeCompare([]byte(adminSecret), []byte(AdminSecretKey)) != 1 {
			return router.ResponseUnauthorized(c, "Invalid admin secret")
		}

		return c.Next()
	}
}

// APIKeyAuth validates the X-API-Key header and stores API key context
func APIKeyAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			return router.ResponseUnauthorized(c, "Missing X-API-Key header")
		}

		ctx := c.UserContext()
		if ctx == nil {
			ctx = context.Background()
		}

		apiKeyRecord, err := pkgWhatsApp.GetAPIKeyByKey(ctx, apiKey)
		if err != nil {
			return router.ResponseUnauthorized(c, "Invalid API key")
		}

		if !apiKeyRecord.IsActive {
			return router.ResponseUnauthorized(c, "API key is inactive")
		}

		// Store API key context in locals
		c.Locals("api_key", apiKeyRecord)
		c.Locals("api_key_id", apiKeyRecord.ID)

		return c.Next()
	}
}

// DeviceAuth validates the JWT token from Authorization header
// Token format: "Bearer <jwt_token>"
// This is a stateless validation - no database hit for every request
// JWT version is checked against database only when needed (optional)
func DeviceAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return router.ResponseUnauthorized(c, "Missing Authorization header")
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return router.ResponseUnauthorized(c, "Invalid Authorization header format. Use: Bearer <token>")
		}

		tokenString := parts[1]
		if tokenString == "" {
			return router.ResponseUnauthorized(c, "Missing token")
		}

		// Validate JWT token (stateless - no DB hit)
		claims, err := ValidateDeviceToken(tokenString)
		if err != nil {
			return router.ResponseUnauthorized(c, "Invalid or expired token")
		}

		// Optional: Verify JWT version against database
		// This adds a DB hit but ensures tokens can be invalidated immediately
		// Comment out the following block if you want pure stateless validation
		ctx := c.UserContext()
		if ctx == nil {
			ctx = context.Background()
		}
		currentVersion, err := pkgWhatsApp.GetDeviceJWTVersion(ctx, claims.DeviceID)
		if err != nil {
			return router.ResponseUnauthorized(c, "Device not found")
		}
		if claims.JWTVersion != currentVersion {
			return router.ResponseUnauthorized(c, "Token has been revoked. Please regenerate a new token.")
		}

		// Store device context in locals (from JWT claims - no DB hit)
		c.Locals("device_id", claims.DeviceID)
		c.Locals("device_jid", claims.JID)
		c.Locals("api_key_id", claims.APIKeyID)
		c.Locals("jwt_version", claims.JWTVersion)

		return c.Next()
	}
}
