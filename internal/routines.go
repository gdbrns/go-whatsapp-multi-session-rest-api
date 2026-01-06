package internal

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

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

	if isHealthCheckEnabled() {
		_, err := cron.AddFunc("0 */5 * * * *", func() {
			if pkgWhatsApp.WhatsAppClientsLen() == 0 {
				return
			}
			pkgWhatsApp.WhatsAppRangeClients(func(jid string, deviceID string, client *whatsmeow.Client) {
				if len(jid) < 4 {
					return
				}
				maskJID := jid[0:len(jid)-4] + "xxxx"
				isConnected := client.IsConnected()
				isLoggedIn := client.IsLoggedIn()
				if !isConnected || !isLoggedIn {
					log.Print(nil).Warn("Client unhealthy: " + maskJID + " (" + deviceID + ")")
					// Sync DB status to disconnected if client is not healthy
					_ = pkgWhatsApp.UpdateDeviceStatus(context.Background(), deviceID, "disconnected")
				} else {
					log.Print(nil).Info("Client healthy: " + maskJID + " (" + deviceID + ")")
					// Sync DB status to active if client is healthy
					_ = pkgWhatsApp.UpdateDeviceStatus(context.Background(), deviceID, "active")
				}
			})
		})
		if err != nil {
			log.Print(nil).WithField("error", err.Error()).Error("Failed to add health check cron job")
		}
	} else {
		log.Print(nil).Info("Health check cron disabled; relying on whatsmeow event handlers")
	}

	if isWAVersionRefreshCronEnabled() {
		spec := getWAVersionRefreshCronSpec()
		force := getWAVersionRefreshCronForce()
		_, err := cron.AddFunc(spec, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			status, refreshed, err := pkgWhatsApp.WhatsAppRefreshWAVersion(ctx, force)
			v := status.CurrentVersion
			versionStr := strconv.FormatUint(uint64(v[0]), 10) + "." + strconv.FormatUint(uint64(v[1]), 10) + "." + strconv.FormatUint(uint64(v[2]), 10)
			if err != nil {
				log.Print(nil).WithField("version", versionStr).WithField("force", force).Error("WA Web version refresh failed: " + err.Error())
				return
			}
			log.Print(nil).WithField("version", versionStr).WithField("refreshed", refreshed).WithField("force", force).Info("WA Web version refresh completed")
		})
		if err != nil {
			log.Print(nil).WithField("error", err.Error()).Error("Failed to add WA Web version refresh cron job")
		} else {
			log.Print(nil).WithField("spec", spec).WithField("force", force).Info("WA Web version refresh cron enabled")
		}
	}

	cron.Start()
}

func isHealthCheckEnabled() bool {
	envValue, ok := os.LookupEnv("WHATSAPP_ENABLE_HEALTH_CHECK_CRON")
	if !ok {
		// Default to true - ensures DB status stays in sync with actual client state
		return true
	}
	enabled, err := strconv.ParseBool(envValue)
	if err != nil {
		log.Print(nil).Warn("Invalid WHATSAPP_ENABLE_HEALTH_CHECK_CRON value; defaulting to enabled")
		return true
	}
	return enabled
}

func isWAVersionRefreshCronEnabled() bool {
	envValue, ok := os.LookupEnv("WHATSAPP_ENABLE_WAVERSION_REFRESH_CRON")
	if !ok {
		return false
	}
	enabled, err := strconv.ParseBool(strings.TrimSpace(envValue))
	if err != nil {
		log.Print(nil).Warn("Invalid WHATSAPP_ENABLE_WAVERSION_REFRESH_CRON value; defaulting to disabled")
		return false
	}
	return enabled
}

func getWAVersionRefreshCronSpec() string {
	// robfig/cron with seconds field (6 parts). Default: daily at 03:00:00.
	spec := strings.TrimSpace(os.Getenv("WHATSAPP_WAVERSION_REFRESH_CRON_SPEC"))
	if spec == "" {
		return "0 0 3 * * *"
	}
	return spec
}

func getWAVersionRefreshCronForce() bool {
	// Default: false (respects WHATSAPP_WAVERSION_REFRESH_MIN_INTERVAL throttling).
	raw := strings.TrimSpace(os.Getenv("WHATSAPP_WAVERSION_REFRESH_CRON_FORCE"))
	if raw == "" {
		return false
	}
	b, err := strconv.ParseBool(raw)
	if err != nil {
		return false
	}
	return b
}
