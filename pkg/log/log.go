package log

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

// LogLevel controls verbosity: "debug", "info", "warn", "error"
// Set via LOG_LEVEL env var. Default is "info".
func init() {
	level := os.Getenv("LOG_LEVEL")
	switch level {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	// For production B2B, consider JSON format for log aggregation
	if os.Getenv("LOG_FORMAT") == "json" {
		logger.Formatter = &logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		}
	} else {
		logger.Formatter = &logrus.TextFormatter{
			TimestampFormat: time.RFC3339,
			FullTimestamp:   true,
			DisableColors:   false,
			ForceColors:     true,
		}
	}
}

func Print(c *fiber.Ctx) *logrus.Entry {
	if c == nil {
		return logger.WithFields(logrus.Fields{})
	}

	remoteIP := c.IP()
	if v := c.Locals("remote_ip"); v != nil {
		if ip, ok := v.(string); ok && ip != "" {
			remoteIP = ip
		}
	}

	fields := logrus.Fields{
		"remote_ip": remoteIP,
		"method":    c.Method(),
		"uri":       c.OriginalURL(),
	}

	if reqID := c.Locals("request_id"); reqID != nil {
		if id, ok := reqID.(string); ok && id != "" {
			fields["request_id"] = id
		}
	}

	// Add device context for B2B multi-device tracing
	if deviceID := c.Locals("device_id"); deviceID != nil {
		fields["device_id"] = deviceID
	}
	if deviceJID := c.Locals("device_jid"); deviceJID != nil {
		fields["device_jid"] = deviceJID
	}

	return logger.WithFields(fields)
}

// PrintWithDevice creates a log entry with device context (for use outside request handlers)
func PrintWithDevice(deviceID, jid string) *logrus.Entry {
	fields := logrus.Fields{}
	if deviceID != "" {
		fields["device_id"] = deviceID
	}
	if jid != "" {
		fields["device_jid"] = jid
	}
	return logger.WithFields(fields)
}

// Session creates a log entry with full session context including operation name
// This is the primary logging function for multi-session operations using fiber context
func Session(c *fiber.Ctx, operation string) *logrus.Entry {
	fields := logrus.Fields{
		"operation": operation,
	}

	if c != nil {
		remoteIP := c.IP()
		if v := c.Locals("remote_ip"); v != nil {
			if ip, ok := v.(string); ok && ip != "" {
				remoteIP = ip
			}
		}
		fields["remote_ip"] = remoteIP
		fields["method"] = c.Method()
		fields["uri"] = c.OriginalURL()

		if reqID := c.Locals("request_id"); reqID != nil {
			if id, ok := reqID.(string); ok && id != "" {
				fields["request_id"] = id
			}
		}

		// Add device context for multi-session tracing
		if deviceID := c.Locals("device_id"); deviceID != nil {
			fields["device_id"] = deviceID
		}
		if deviceJID := c.Locals("device_jid"); deviceJID != nil {
			fields["device_jid"] = deviceJID
		}
		// Add API key context for customer tracing
		if apiKeyID := c.Locals("api_key_id"); apiKeyID != nil {
			fields["api_key_id"] = apiKeyID
		}
	}

	return logger.WithFields(fields)
}

// SessionWithDevice creates a log entry with explicit device context and operation name
// Use this when device context is extracted from fiber.Ctx locals
func SessionWithDevice(deviceID, jid, operation string) *logrus.Entry {
	fields := logrus.Fields{
		"operation": operation,
	}
	if deviceID != "" {
		fields["device_id"] = deviceID
	}
	if jid != "" {
		fields["device_jid"] = jid
	}
	return logger.WithFields(fields)
}

// DeviceOp creates a log entry for device-specific operations with explicit device context
func DeviceOp(deviceID, jid, operation string) *logrus.Entry {
	fields := logrus.Fields{
		"operation": operation,
	}
	if deviceID != "" {
		fields["device_id"] = deviceID
	}
	if jid != "" {
		fields["device_jid"] = jid
	}
	return logger.WithFields(fields)
}

// DeviceOpCtx builds a device-scoped log entry using fiber context locals
// (device_id, device_jid, request_id).
func DeviceOpCtx(c *fiber.Ctx, operation string) *logrus.Entry {
	fields := logrus.Fields{
		"operation": operation,
	}
	if c != nil {
		if reqID := c.Locals("request_id"); reqID != nil {
			if id, ok := reqID.(string); ok && id != "" {
				fields["request_id"] = id
			}
		}
		if deviceID := c.Locals("device_id"); deviceID != nil {
			fields["device_id"] = deviceID
		}
		if jid := c.Locals("device_jid"); jid != nil {
			fields["device_jid"] = jid
		}
	}
	return logger.WithFields(fields)
}

// AdminOp creates a log entry for admin operations
func AdminOp(c *fiber.Ctx, operation string) *logrus.Entry {
	fields := logrus.Fields{
		"operation": operation,
		"scope":     "admin",
	}

	if c != nil {
		remoteIP := c.IP()
		if v := c.Locals("remote_ip"); v != nil {
			if ip, ok := v.(string); ok && ip != "" {
				remoteIP = ip
			}
		}
		fields["remote_ip"] = remoteIP
		fields["method"] = c.Method()
		fields["uri"] = c.OriginalURL()
		if reqID := c.Locals("request_id"); reqID != nil {
			if id, ok := reqID.(string); ok && id != "" {
				fields["request_id"] = id
			}
		}
	}

	return logger.WithFields(fields)
}

// WebhookOp creates a log entry for webhook operations
func WebhookOp(deviceID, jid, operation string, webhookID int64) *logrus.Entry {
	fields := logrus.Fields{
		"operation":  operation,
		"scope":      "webhook",
		"webhook_id": webhookID,
	}
	if deviceID != "" {
		fields["device_id"] = deviceID
	}
	if jid != "" {
		fields["device_jid"] = jid
	}
	return logger.WithFields(fields)
}

// MessageOp creates a log entry for messaging operations with target info
func MessageOp(deviceID, jid, operation, targetJID string) *logrus.Entry {
	fields := logrus.Fields{
		"operation": operation,
		"scope":     "messaging",
	}
	if deviceID != "" {
		fields["device_id"] = deviceID
	}
	if jid != "" {
		fields["device_jid"] = jid
	}
	if targetJID != "" {
		fields["target_jid"] = targetJID
	}
	return logger.WithFields(fields)
}

// MessageOpCtx builds a messaging log entry with request context.
func MessageOpCtx(c *fiber.Ctx, operation, targetJID string) *logrus.Entry {
	fields := logrus.Fields{
		"operation": operation,
		"scope":     "messaging",
	}
	if c != nil {
		if reqID := c.Locals("request_id"); reqID != nil {
			if id, ok := reqID.(string); ok && id != "" {
				fields["request_id"] = id
			}
		}
		if deviceID := c.Locals("device_id"); deviceID != nil {
			fields["device_id"] = deviceID
		}
		if jid := c.Locals("device_jid"); jid != nil {
			fields["device_jid"] = jid
		}
	}
	if targetJID != "" {
		fields["target_jid"] = targetJID
	}
	return logger.WithFields(fields)
}

// GroupOp creates a log entry for group operations
func GroupOp(deviceID, jid, operation, groupJID string) *logrus.Entry {
	fields := logrus.Fields{
		"operation": operation,
		"scope":     "group",
	}
	if deviceID != "" {
		fields["device_id"] = deviceID
	}
	if jid != "" {
		fields["device_jid"] = jid
	}
	if groupJID != "" {
		fields["group_jid"] = groupJID
	}
	return logger.WithFields(fields)
}

// AuthOp creates a log entry for authentication operations
func AuthOp(c *fiber.Ctx, operation, deviceID string) *logrus.Entry {
	fields := logrus.Fields{
		"operation": operation,
		"scope":     "auth",
	}
	if deviceID != "" {
		fields["device_id"] = deviceID
	}
	if c != nil {
		remoteIP := c.IP()
		if v := c.Locals("remote_ip"); v != nil {
			if ip, ok := v.(string); ok && ip != "" {
				remoteIP = ip
			}
		}
		fields["remote_ip"] = remoteIP
	}
	return logger.WithFields(fields)
}
