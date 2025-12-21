package whatsapp

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"golang.org/x/sync/singleflight"
)

var ErrWAVersionOutdatedForQR = errors.New("whatsapp client version is outdated for QR pairing")

type WAVersionRefreshStatus struct {
	CurrentVersion store.WAVersionContainer `json:"current_version"`
	LastRefreshed  *time.Time              `json:"last_refreshed,omitempty"`
	LastError      string                  `json:"last_error,omitempty"`
}

var (
	waVersionRefreshGroup singleflight.Group

	waVersionRefreshMu       sync.RWMutex
	waVersionLastRefreshedAt *time.Time
	waVersionLastError       string
)

func getWAVersionRefreshMinInterval() time.Duration {
	// Default: conservative (avoid hammering the endpoint)
	const defaultInterval = 10 * time.Minute
	raw := strings.TrimSpace(os.Getenv("WHATSAPP_WAVERSION_REFRESH_MIN_INTERVAL"))
	if raw == "" {
		return defaultInterval
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d < 0 {
		return defaultInterval
	}
	return d
}

func WhatsAppGetWAVersionRefreshStatus() WAVersionRefreshStatus {
	waVersionRefreshMu.RLock()
	defer waVersionRefreshMu.RUnlock()

	current := store.GetWAVersion()
	var last *time.Time
	if waVersionLastRefreshedAt != nil {
		t := *waVersionLastRefreshedAt
		last = &t
	}

	return WAVersionRefreshStatus{
		CurrentVersion: current,
		LastRefreshed:  last,
		LastError:      waVersionLastError,
	}
}

// WhatsAppRefreshWAVersion fetches the latest WhatsApp Web version and applies it globally via store.SetWAVersion.
// If force=false, it will be throttled by WHATSAPP_WAVERSION_REFRESH_MIN_INTERVAL (default 10m).
func WhatsAppRefreshWAVersion(ctx context.Context, force bool) (WAVersionRefreshStatus, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	minInterval := getWAVersionRefreshMinInterval()
	if !force && minInterval > 0 {
		waVersionRefreshMu.RLock()
		last := waVersionLastRefreshedAt
		waVersionRefreshMu.RUnlock()
		if last != nil && time.Since(*last) < minInterval {
			return WhatsAppGetWAVersionRefreshStatus(), false, nil
		}
	}

	res, err, _ := waVersionRefreshGroup.Do("refresh", func() (interface{}, error) {
		httpClient := &http.Client{Timeout: 15 * time.Second}
		latest, err := whatsmeow.GetLatestVersion(ctx, httpClient)
		if err != nil {
			waVersionRefreshMu.Lock()
			now := time.Now()
			waVersionLastRefreshedAt = &now
			waVersionLastError = err.Error()
			waVersionRefreshMu.Unlock()
			return nil, err
		}
		if latest == nil {
			err := errors.New("latest WhatsApp Web version is nil")
			waVersionRefreshMu.Lock()
			now := time.Now()
			waVersionLastRefreshedAt = &now
			waVersionLastError = err.Error()
			waVersionRefreshMu.Unlock()
			return nil, err
		}

		store.SetWAVersion(*latest)

		waVersionRefreshMu.Lock()
		now := time.Now()
		waVersionLastRefreshedAt = &now
		waVersionLastError = ""
		waVersionRefreshMu.Unlock()

		return store.GetWAVersion(), nil
	})
	if err != nil {
		return WhatsAppGetWAVersionRefreshStatus(), true, err
	}

	v, _ := res.(store.WAVersionContainer)
	_ = v
	return WhatsAppGetWAVersionRefreshStatus(), true, nil
}
