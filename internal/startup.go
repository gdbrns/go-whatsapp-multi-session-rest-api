package internal

import (
	"context"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/log"
	pkgWhatsApp "github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/whatsapp"
)

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

	var restored, reconnected, failed int

	for _, device := range devices {
		if device.ID == nil {
			continue
		}
		jid := pkgWhatsApp.WhatsAppDecomposeJID(device.ID.User)
		if len(jid) < 4 {
			continue
		}
		deviceID, errDevice := getDeviceID(device.ID.String())
		if errDevice != nil {
			log.Print(nil).Warn("Device mapping not found for JID " + device.ID.String() + ", skipping restore")
			continue
		}
		maskJID := jid[0:len(jid)-4] + "xxxx"
		log.Print(nil).Info("Restoring WhatsApp Client for " + maskJID + " (" + deviceID + ")")
		pkgWhatsApp.WhatsAppInitClient(device, jid, deviceID)
		restored++
		err = pkgWhatsApp.WhatsAppReconnect(jid, deviceID)
		if err != nil {
			log.Print(nil).Warn("Failed to reconnect " + maskJID + ": " + err.Error())
			failed++
		}
		if err == nil {
			reconnected++
		}
	}

	log.Print(nil).WithField("restored", restored).WithField("reconnected", reconnected).WithField("failed", failed).Info("Startup reconnect pass complete")
}

func getDeviceID(storeDeviceID string) (string, error) {
	return pkgWhatsApp.GetDeviceIDByJID(context.Background(), storeDeviceID)
}
