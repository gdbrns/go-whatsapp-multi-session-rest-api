package log

import (
	"fmt"
	"os"
	"strings"
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

	// Ultra-short format for clean backend logs
	if os.Getenv("LOG_FORMAT") == "json" {
		logger.Formatter = &logrus.JSONFormatter{
			TimestampFormat: "15:04:05",
		}
	} else {
		logger.Formatter = &logrus.TextFormatter{
			TimestampFormat: "15:04:05",
			FullTimestamp:   true,
			DisableColors:   false,
			ForceColors:     true,
		}
	}
}

// shortID returns last 6 chars of ID for compact logging
func shortID(id string) string {
	if len(id) <= 6 {
		return id
	}
	return id[len(id)-6:]
}

// shortJID returns phone number only from JID (e.g., "628xxx" from "628xxx@s.whatsapp.net")
func shortJID(jid string) string {
	if idx := strings.Index(jid, "@"); idx > 0 {
		phone := jid[:idx]
		if len(phone) > 8 {
			return phone[:4] + ".." + phone[len(phone)-4:]
		}
		return phone
	}
	if len(jid) > 8 {
		return jid[:4] + ".." + jid[len(jid)-4:]
	}
	return jid
}

// Evt logs an event with ultra-short format: [scope] action device target
func Evt(scope, action, deviceID string, extra ...string) {
	msg := fmt.Sprintf("[%s] %s d:%s", scope, action, shortID(deviceID))
	if len(extra) > 0 {
		msg += " " + strings.Join(extra, " ")
	}
	logger.Info(msg)
}

// EvtOK logs successful event
func EvtOK(scope, action, deviceID string, extra ...string) {
	msg := fmt.Sprintf("[%s] ✓ %s d:%s", scope, action, shortID(deviceID))
	if len(extra) > 0 {
		msg += " " + strings.Join(extra, " ")
	}
	logger.Info(msg)
}

// EvtErr logs error event
func EvtErr(scope, action, deviceID string, err error) {
	logger.Error(fmt.Sprintf("[%s] ✗ %s d:%s err:%v", scope, action, shortID(deviceID), err))
}

// WH logs webhook dispatch with ACK
func WH(eventType, deviceID string, webhookCount int) {
	if webhookCount > 0 {
		logger.Info(fmt.Sprintf("[wh] → %s d:%s (%d hooks)", eventType, shortID(deviceID), webhookCount))
	}
}

// WHACK logs webhook delivery ACK
func WHACK(eventType, deviceID string, webhookID int64, success bool, attempt int) {
	status := "✓"
	if !success {
		status = "✗"
	}
	logger.Info(fmt.Sprintf("[wh] %s %s d:%s wh:%d att:%d", status, eventType, shortID(deviceID), webhookID, attempt))
}

// Conn logs connection events
func Conn(action, deviceID, jid string) {
	logger.Info(fmt.Sprintf("[conn] %s d:%s j:%s", action, shortID(deviceID), shortJID(jid)))
}

// Msg logs message events
func Msg(action, deviceID, msgID, target string) {
	logger.Info(fmt.Sprintf("[msg] %s d:%s m:%s t:%s", action, shortID(deviceID), shortID(msgID), shortJID(target)))
}

// API logs API request/response
func API(method, path string, status int, latency time.Duration) {
	logger.Info(fmt.Sprintf("[api] %s %s %d %dms", method, path, status, latency.Milliseconds()))
}

// Grp logs group events
func Grp(action, deviceID, groupJID string) {
	logger.Info(fmt.Sprintf("[grp] %s d:%s g:%s", action, shortID(deviceID), shortJID(groupJID)))
}

// Call logs call events
func Call(action, deviceID, callID, from string) {
	logger.Info(fmt.Sprintf("[call] %s d:%s c:%s f:%s", action, shortID(deviceID), shortID(callID), shortJID(from)))
}

// Sys logs system events
func Sys(action string, extra ...string) {
	msg := fmt.Sprintf("[sys] %s", action)
	if len(extra) > 0 {
		msg += " " + strings.Join(extra, " ")
	}
	logger.Info(msg)
}

// SysErr logs system errors
func SysErr(action string, err error) {
	logger.Error(fmt.Sprintf("[sys] ✗ %s err:%v", action, err))
}

// Debug logs debug info (only if LOG_LEVEL=debug)
func Debug(scope, msg string) {
	logger.Debug(fmt.Sprintf("[%s] %s", scope, msg))
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
