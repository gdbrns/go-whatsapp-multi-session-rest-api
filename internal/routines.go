package internal

import (
	"os"
	"strconv"

	"github.com/robfig/cron/v3"
	"go.mau.fi/whatsmeow"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/log"
	pkgWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/whatsapp"
)

// ❌ YOUR ADDITION: Cron-based session validation and cleanup
// whatsmeow provides client.IsConnected() and client.IsLoggedIn() for basic validation,
// but doesn't automatically clean up mismatched sessions or perform periodic health checks
func Routines(cron *cron.Cron) {
	log.Print(nil).Info("Running Routine Tasks")

	if !isHealthCheckEnabled() {
		log.Print(nil).Info("Health check cron disabled; relying on whatsmeow event handlers")
		cron.Start()
		return
	}

	cron.AddFunc("0 */5 * * * *", func() {
		if pkgWhatsApp.WhatsAppClientsLen() == 0 {
			return
		}
		pkgWhatsApp.WhatsAppRangeClients(func(jid string, deviceID string, client *whatsmeow.Client) {
			if len(jid) < 4 {
				return
			}
			maskJID := jid[0:len(jid)-4] + "xxxx"
			if !client.IsConnected() || !client.IsLoggedIn() {
				log.Print(nil).Warn("Client unhealthy: " + maskJID + " (" + deviceID + ")")
			} else {
				log.Print(nil).Info("Client healthy: " + maskJID + " (" + deviceID + ")")
			}
		})
	})

	cron.Start()
}

func isHealthCheckEnabled() bool {
	envValue, ok := os.LookupEnv("WHATSAPP_ENABLE_HEALTH_CHECK_CRON")
	if !ok {
		return false
	}
	enabled, err := strconv.ParseBool(envValue)
	if err != nil {
		log.Print(nil).Warn("Invalid WHATSAPP_ENABLE_HEALTH_CHECK_CRON value; defaulting to disabled")
		return false
	}
	return enabled
}
