package webhook

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/env"
)

type Store struct {
	db             *sql.DB
	cacheMu        sync.RWMutex
	activeCache    map[string]activeCacheEntry
	activeCacheTTL time.Duration
}

type activeCacheEntry struct {
	webhooks  []WebhookConfig
	expiresAt time.Time
}

func NewStore(db *sql.DB) *Store {
	ttlSeconds := env.GetEnvIntOrDefault("WEBHOOK_CACHE_TTL_SECONDS", 15)
	if ttlSeconds < 0 {
		ttlSeconds = 0
	}
	return &Store{
		db:             db,
		activeCache:    make(map[string]activeCacheEntry),
		activeCacheTTL: time.Duration(ttlSeconds) * time.Second,
	}
}

func (s *Store) getActiveCache(deviceID string) ([]WebhookConfig, bool) {
	if s.activeCacheTTL <= 0 {
		return nil, false
	}
	s.cacheMu.RLock()
	entry, ok := s.activeCache[deviceID]
	s.cacheMu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		s.cacheMu.Lock()
		delete(s.activeCache, deviceID)
		s.cacheMu.Unlock()
		return nil, false
	}
	return entry.webhooks, true
}

func (s *Store) setActiveCache(deviceID string, webhooks []WebhookConfig) {
	if s.activeCacheTTL <= 0 {
		return
	}
	s.cacheMu.Lock()
	s.activeCache[deviceID] = activeCacheEntry{
		webhooks:  webhooks,
		expiresAt: time.Now().Add(s.activeCacheTTL),
	}
	s.cacheMu.Unlock()
}

func (s *Store) invalidateActiveCache(deviceID string) {
	if s.activeCacheTTL <= 0 {
		return
	}
	s.cacheMu.Lock()
	delete(s.activeCache, deviceID)
	s.cacheMu.Unlock()
}

func (s *Store) GetAllWebhooks(ctx context.Context, deviceID string) ([]WebhookConfig, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, device_id, url, secret, events, active, created_at, updated_at
		FROM wa_webhooks
		WHERE device_id = $1
	`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []WebhookConfig
	for rows.Next() {
		var w WebhookConfig
		var eventsJSON []byte
		err := rows.Scan(&w.ID, &w.DeviceID, &w.URL, &w.Secret, &eventsJSON, &w.Active, &w.CreatedAt, &w.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(eventsJSON, &w.Events); err != nil {
			return nil, err
		}
		webhooks = append(webhooks, w)
	}
	return webhooks, rows.Err()
}

func (s *Store) GetActiveWebhooks(ctx context.Context, deviceID string) ([]WebhookConfig, error) {
	if cached, ok := s.getActiveCache(deviceID); ok {
		return cached, nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, device_id, url, secret, events, active, created_at, updated_at
		FROM wa_webhooks
		WHERE device_id = $1 AND active = TRUE
	`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []WebhookConfig
	for rows.Next() {
		var w WebhookConfig
		var eventsJSON []byte
		err := rows.Scan(&w.ID, &w.DeviceID, &w.URL, &w.Secret, &eventsJSON, &w.Active, &w.CreatedAt, &w.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(eventsJSON, &w.Events); err != nil {
			return nil, err
		}
		webhooks = append(webhooks, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	s.setActiveCache(deviceID, webhooks)
	return webhooks, nil
}

func (s *Store) GetWebhook(ctx context.Context, webhookID int64, deviceID string) (*WebhookConfig, error) {
	var w WebhookConfig
	var eventsJSON []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT id, device_id, url, secret, events, active, created_at, updated_at
		FROM wa_webhooks
		WHERE id = $1 AND device_id = $2
	`, webhookID, deviceID).Scan(&w.ID, &w.DeviceID, &w.URL, &w.Secret, &eventsJSON, &w.Active, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(eventsJSON, &w.Events); err != nil {
		return nil, err
	}
	return &w, nil
}

func (s *Store) CreateWebhook(ctx context.Context, deviceID, url, secret string, events []EventType) (int64, error) {
	eventsJSON, err := json.Marshal(events)
	if err != nil {
		return 0, err
	}

	var id int64
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO wa_webhooks (device_id, url, secret, events, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4::jsonb, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, deviceID, url, secret, string(eventsJSON)).Scan(&id)
	if err == nil {
		s.invalidateActiveCache(deviceID)
	}
	return id, err
}

func (s *Store) UpdateWebhook(ctx context.Context, webhookID int64, deviceID, url, secret string, events []EventType, active bool) error {
	eventsJSON, err := json.Marshal(events)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE wa_webhooks
		SET url = $1, secret = $2, events = $3::jsonb, active = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5 AND device_id = $6
	`, url, secret, string(eventsJSON), active, webhookID, deviceID)
	if err == nil {
		s.invalidateActiveCache(deviceID)
	}
	return err
}

func (s *Store) DeleteWebhook(ctx context.Context, webhookID int64, deviceID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM wa_webhooks WHERE id = $1 AND device_id = $2
	`, webhookID, deviceID)
	if err == nil {
		s.invalidateActiveCache(deviceID)
	}
	return err
}

func (s *Store) LogDelivery(ctx context.Context, webhookID int64, eventType EventType, status DeliveryStatus, attemptCount int, lastError string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO wa_webhook_deliveries (webhook_id, event_type, status, attempt_count, last_error, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, webhookID, eventType, status, attemptCount, lastError)
	return err
}

func (s *Store) UpdateDeliveryStatus(ctx context.Context, deliveryID int64, status DeliveryStatus, attemptCount int, lastError string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE wa_webhook_deliveries
		SET status = $1, attempt_count = $2, last_error = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4
	`, status, attemptCount, lastError, deliveryID)
	return err
}

func (s *Store) GetDeliveryLogs(ctx context.Context, webhookID int64, limit int) ([]DeliveryLog, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, webhook_id, event_type, status, attempt_count, last_error, created_at, updated_at
		FROM wa_webhook_deliveries
		WHERE webhook_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, webhookID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []DeliveryLog
	for rows.Next() {
		var log DeliveryLog
		var lastError sql.NullString
		err := rows.Scan(&log.ID, &log.WebhookID, &log.EventType, &log.Status, &log.AttemptCount, &lastError, &log.CreatedAt, &log.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if lastError.Valid {
			log.LastError = lastError.String
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}
