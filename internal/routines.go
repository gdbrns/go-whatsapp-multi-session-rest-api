package internal

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/log"
	pkgWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/whatsapp"
)

// Cron-based session validation, cleanup, and auto-reconnection
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
					log.Print(nil).Warn("Client unhealthy: " + maskJID + " (" + deviceID + "), attempting reconnect...")

					// Attempt auto-reconnect if client has valid store
					if client.Store != nil && client.Store.ID != nil {
						if isConnected && !isLoggedIn {
							// Connected but not logged in — handshake stuck.
							// Connect() would return ErrAlreadyConnected.
							// Use ResetConnection() to force a clean restart.
							client.ResetConnection()
						} else {
							// Not connected at all — safe to call Connect()
							reconnectErr := client.Connect()
							if reconnectErr == nil {
								log.Print(nil).Info("Client reconnected successfully: " + maskJID + " (" + deviceID + ")")
								_ = pkgWhatsApp.UpdateDeviceStatus(context.Background(), deviceID, "active")
								return
							}
							log.Print(nil).WithField("error", reconnectErr.Error()).Warn("Client reconnect failed: " + maskJID + " (" + deviceID + ")")
						}
					}

					// Sync DB status to disconnected if reconnect failed or no valid store
					_ = pkgWhatsApp.UpdateDeviceStatus(context.Background(), deviceID, "disconnected")
				} else {
					log.Print(nil).Debug("Client healthy: " + maskJID + " (" + deviceID + ")")
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

	// Device recovery cron - attempts to reconnect devices from DB that are not in memory
	// Runs every 10 minutes by default, enabled by default
	// Now uses GetDevicesNeedingRecovery which includes BOTH 'active' and 'disconnected' status devices
	// This fixes the issue where devices with 'active' status in DB but not in memory after restart
	if isDeviceRecoveryCronEnabled() {
		spec := getDeviceRecoveryCronSpec()
		_, err := cron.AddFunc(spec, func() {
			ctx := context.Background()
			// Use GetDevicesNeedingRecovery to get ALL devices that should be in memory
			devices, err := pkgWhatsApp.GetDevicesNeedingRecovery(ctx)
			if err != nil {
				log.Print(nil).WithField("error", err.Error()).Error("Failed to get devices for recovery")
				return
			}

			if len(devices) == 0 {
				return
			}

			recovered := 0
			skipped := 0
			for _, device := range devices {
				if device.WhatsMeowJID == "" {
					continue
				}

				jid := pkgWhatsApp.WhatsAppDecomposeJID(extractJIDUser(device.WhatsMeowJID))
				if len(jid) < 4 {
					continue
				}

				// Check if client already exists in memory AND is connected
				if pkgWhatsApp.WhatsAppClientExists(jid, device.DeviceID) {
					client := pkgWhatsApp.WhatsAppGetClient(jid, device.DeviceID)
					if client != nil && client.Store != nil && client.Store.ID != nil {
						if client.IsConnected() {
							// Already connected, skip
							skipped++
							continue
						}
						// Client exists but not connected - try to reconnect
						err := client.Connect()
						if err == nil {
							log.Print(nil).Info("Recovered device (reconnect): " + device.DeviceID)
							_ = pkgWhatsApp.UpdateDeviceStatus(ctx, device.DeviceID, "active")
							recovered++
						} else {
							log.Print(nil).WithField("error", err.Error()).Warn("Failed to reconnect device: " + device.DeviceID)
							_ = pkgWhatsApp.UpdateDeviceStatus(ctx, device.DeviceID, "disconnected")
						}
					}
					continue
				}

				// Client doesn't exist in memory - try to restore from whatsmeow store
				// This is the key fix: devices that were "active" before restart but not in memory
				parsedJID, parseErr := parseJID(device.WhatsMeowJID)
				if parseErr != nil {
					log.Print(nil).WithField("device_id", device.DeviceID).WithField("jid", device.WhatsMeowJID).Warn("Invalid whatsmeow JID, skipping recovery")
					continue
				}
				storeDevice, err := pkgWhatsApp.WhatsAppDatastore.GetDevice(ctx, parsedJID)
				if err != nil || storeDevice == nil {
					log.Print(nil).WithField("device_id", device.DeviceID).Warn("Device not found in whatsmeow store, marking as logged_out")
					_ = pkgWhatsApp.UpdateDeviceStatus(ctx, device.DeviceID, "logged_out")
					pkgWhatsApp.CleanupDeviceRateLimiter(device.DeviceID)
					continue
				}

				// Initialize and connect
				log.Print(nil).WithField("device_id", device.DeviceID).Info("Restoring device from whatsmeow store")
				pkgWhatsApp.WhatsAppInitClient(storeDevice, jid, device.DeviceID)
				client := pkgWhatsApp.WhatsAppGetClient(jid, device.DeviceID)
				if client != nil && client.Store != nil && client.Store.ID != nil {
					err := client.Connect()
					if err == nil {
						log.Print(nil).Info("Recovered device (restore+connect): " + device.DeviceID)
						_ = pkgWhatsApp.UpdateDeviceStatus(ctx, device.DeviceID, "active")
						recovered++
					} else {
						log.Print(nil).WithField("error", err.Error()).Warn("Failed to connect restored device: " + device.DeviceID)
						_ = pkgWhatsApp.UpdateDeviceStatus(ctx, device.DeviceID, "disconnected")
					}
				}
			}

			if recovered > 0 || len(devices) > skipped {
				log.Print(nil).WithField("recovered", recovered).WithField("skipped", skipped).WithField("total", len(devices)).Info("Device recovery completed")
			}
		})
		if err != nil {
			log.Print(nil).WithField("error", err.Error()).Error("Failed to add device recovery cron job")
		} else {
			log.Print(nil).WithField("spec", spec).Info("Device recovery cron enabled")
		}
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

	// Webhook delivery log cleanup cron — prevents unbounded wa_webhook_deliveries table growth (#5)
	// Runs daily at 04:00, deletes deliveries older than 7 days by default
	if whe := pkgWhatsApp.GetWebhookEngine(); whe != nil {
		retentionDays := 7
		if raw, ok := os.LookupEnv("WEBHOOK_DELIVERY_RETENTION_DAYS"); ok {
			if v, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && v > 0 {
				retentionDays = v
			}
		}
		retention := time.Duration(retentionDays) * 24 * time.Hour
		_, err := cron.AddFunc("0 0 4 * * *", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			deleted, err := whe.Store().CleanupOldDeliveries(ctx, retention)
			if err != nil {
				log.Print(nil).WithField("error", err.Error()).Error("Failed to cleanup old webhook deliveries")
				return
			}
			if deleted > 0 {
				log.Print(nil).WithField("deleted", deleted).WithField("retention_days", retentionDays).Info("Webhook delivery log cleanup completed")
			}
		})
		if err != nil {
			log.Print(nil).WithField("error", err.Error()).Error("Failed to add webhook cleanup cron job")
		} else {
			log.Print(nil).WithField("retention_days", retentionDays).Info("Webhook delivery cleanup cron enabled")
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
		// Default to true - keeps WhatsApp Web version up-to-date to prevent connection issues
		return true
	}
	enabled, err := strconv.ParseBool(strings.TrimSpace(envValue))
	if err != nil {
		log.Print(nil).Warn("Invalid WHATSAPP_ENABLE_WAVERSION_REFRESH_CRON value; defaulting to enabled")
		return true
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
	// Default: true - ensures version refresh always runs on schedule
	raw := strings.TrimSpace(os.Getenv("WHATSAPP_WAVERSION_REFRESH_CRON_FORCE"))
	if raw == "" {
		return true
	}
	b, err := strconv.ParseBool(raw)
	if err != nil {
		return true
	}
	return b
}

func isDeviceRecoveryCronEnabled() bool {
	envValue, ok := os.LookupEnv("WHATSAPP_ENABLE_DEVICE_RECOVERY_CRON")
	if !ok {
		// Default to true - ensures disconnected devices are automatically recovered
		return true
	}
	enabled, err := strconv.ParseBool(strings.TrimSpace(envValue))
	if err != nil {
		log.Print(nil).Warn("Invalid WHATSAPP_ENABLE_DEVICE_RECOVERY_CRON value; defaulting to enabled")
		return true
	}
	return enabled
}

func getDeviceRecoveryCronSpec() string {
	// robfig/cron with seconds field (6 parts). Default: every 2 minutes for faster recovery after restart
	spec := strings.TrimSpace(os.Getenv("WHATSAPP_DEVICE_RECOVERY_CRON_SPEC"))
	if spec == "" {
		return "0 */2 * * * *"
	}
	return spec
}

// extractJIDUser extracts the user part from a full JID string (e.g., "1234567890:12@s.whatsapp.net" -> "1234567890")
func extractJIDUser(fullJID string) string {
	// Remove the server part first
	if idx := strings.Index(fullJID, "@"); idx > 0 {
		fullJID = fullJID[:idx]
	}
	// Remove the device part if present
	if idx := strings.Index(fullJID, ":"); idx > 0 {
		fullJID = fullJID[:idx]
	}
	return fullJID
}

// parseJID parses a JID string into types.JID
func parseJID(jidStr string) (types.JID, error) {
	jid, err := types.ParseJID(jidStr)
	if err != nil || jid.User == "" {
		return types.EmptyJID, errors.New("invalid JID")
	}
	return jid, nil
}
