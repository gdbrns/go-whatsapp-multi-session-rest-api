package internal

import (
	"context"
	mathrand "math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/log"
	pkgWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/whatsapp"
	"go.mau.fi/whatsmeow/store"
)

func jitterSleep(max time.Duration) {
	if max <= 0 {
		return
	}
	ms := mathrand.Int64N(max.Milliseconds() + 1)
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func reconnectWithRetry(jid string, deviceID string, retries int, baseBackoff time.Duration, maxBackoff time.Duration) error {
	if retries <= 1 {
		return pkgWhatsApp.WhatsAppReconnect(jid, deviceID)
	}
	if baseBackoff <= 0 {
		baseBackoff = 2 * time.Second
	}
	if maxBackoff <= 0 {
		maxBackoff = 30 * time.Second
	}

	var lastErr error
	for attempt := 1; attempt <= retries; attempt++ {
		lastErr = pkgWhatsApp.WhatsAppReconnect(jid, deviceID)
		if lastErr == nil {
			return nil
		}

		// Exponential backoff with small jitter.
		backoff := baseBackoff * time.Duration(1<<(attempt-1))
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
		jitter := time.Duration(mathrand.Int64N(int64(500*time.Millisecond) + 1))
		time.Sleep(backoff + jitter)
	}
	return lastErr
}

func Startup() {
	log.Print(nil).Info("Running Startup Tasks")

	ctx := context.Background()

	// Start background cache cleanup for multi-device memory management
	pkgWhatsApp.StartCacheCleanup()

	if err := pkgWhatsApp.SyncDeviceRoutings(ctx); err != nil {
		log.Print(nil).Error("Failed to sync device routings: " + err.Error())
	}

	devices, err := pkgWhatsApp.WhatsAppDatastore.GetAllDevices(ctx)
	if err != nil {
		log.Print(nil).Error("Failed to Load WhatsApp Client Devices from Datastore")
		return
	}

	maxConcurrent := pkgWhatsApp.ParseOptionalInt("WHATSAPP_STARTUP_RECONNECT_CONCURRENCY", 10, 1)
	jitterMax := pkgWhatsApp.ParseOptionalDuration("WHATSAPP_STARTUP_RECONNECT_JITTER_MAX", 5*time.Second)
	retries := pkgWhatsApp.ParseOptionalInt("WHATSAPP_STARTUP_RECONNECT_RETRIES", 5, 1) // Default 5 for better recovery
	baseBackoff := pkgWhatsApp.ParseOptionalDuration("WHATSAPP_STARTUP_RECONNECT_BACKOFF_BASE", 2*time.Second)
	maxBackoff := pkgWhatsApp.ParseOptionalDuration("WHATSAPP_STARTUP_RECONNECT_BACKOFF_MAX", 30*time.Second)

	var restored, reconnected, failed int64
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, device := range devices {
		if device == nil || device.ID == nil {
			continue
		}
		dev := device
		jid := pkgWhatsApp.WhatsAppDecomposeJID(device.ID.User)
		if len(jid) < 4 {
			continue
		}
		deviceID, errDevice := getDeviceID(device.ID.String())
		if errDevice != nil {
			fallbackID, fallbackErr := pkgWhatsApp.GetDeviceIDByStoreJID(ctx, device.ID.String())
			if fallbackErr != nil {
				log.Print(nil).Warn("Device mapping not found for JID " + device.ID.String() + ", skipping restore")
				continue
			}
			deviceID = fallbackID
			_ = pkgWhatsApp.SaveDeviceRouting(ctx, deviceID, device.ID.String())
		}
		maskJID := jid[0:len(jid)-4] + "xxxx"

		wg.Add(1)
		sem <- struct{}{}
		go func(dev *store.Device, jidVal string, deviceIDVal string, masked string) {
			defer wg.Done()
			defer func() { <-sem }()

			jitterSleep(jitterMax)
			log.Print(nil).Info("Restoring WhatsApp Client for " + masked + " (" + deviceIDVal + ")")

			// Init client before attempting reconnect.
			// The underlying function is idempotent and will no-op if client already exists.
			pkgWhatsApp.WhatsAppInitClient(dev, jidVal, deviceIDVal)
			atomic.AddInt64(&restored, 1)

			err := reconnectWithRetry(jidVal, deviceIDVal, retries, baseBackoff, maxBackoff)
			if err != nil {
				log.Print(nil).Warn("Failed to reconnect " + masked + ": " + err.Error())
				// Mark as disconnected in DB so health check can retry later
				_ = pkgWhatsApp.UpdateDeviceStatus(context.Background(), deviceIDVal, "disconnected")
				atomic.AddInt64(&failed, 1)
				return
			}
			// Mark as active in DB
			_ = pkgWhatsApp.UpdateDeviceStatus(context.Background(), deviceIDVal, "active")
			atomic.AddInt64(&reconnected, 1)
		}(dev, jid, deviceID, maskJID)
	}

	wg.Wait()
	log.Print(nil).
		WithField("restored", restored).
		WithField("reconnected", reconnected).
		WithField("failed", failed).
		WithField("concurrency", maxConcurrent).
		WithField("retries", retries).
		Info("Startup reconnect pass complete")
}
func getDeviceID(storeDeviceID string) (string, error) {
	return pkgWhatsApp.GetDeviceIDByJID(context.Background(), storeDeviceID)
}
