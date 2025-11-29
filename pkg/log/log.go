package log

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

func Print(c *fiber.Ctx) *logrus.Entry {
	logger.Formatter = &logrus.TextFormatter{
		TimestampFormat: time.RFC3339,
		FullTimestamp:   true,
		DisableColors:   false,
		ForceColors:     true,
	}

	if c == nil {
		return logger.WithFields(logrus.Fields{})
	}

	remoteIP := c.IP()
	if v := c.Locals("remote_ip"); v != nil {
		if ip, ok := v.(string); ok && ip != "" {
			remoteIP = ip
		}
	}
	return logger.WithFields(logrus.Fields{
		"remote_ip": remoteIP,
		"method":    c.Method(),
		"uri":       c.OriginalURL(),
	})
}
