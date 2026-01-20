package whatsapp

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

// APIKey represents a customer API key
type APIKey struct {
	ID               int64      `json:"id"`
	APIKey           string     `json:"api_key"`
	CustomerName     string     `json:"customer_name"`
	CustomerEmail    string     `json:"customer_email"`
	CustomerPhone    string     `json:"customer_phone"`
	MaxDevices       int        `json:"max_devices"`
	RateLimitPerHour int        `json:"rate_limit_per_hour"`
	IsActive         bool       `json:"is_active"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        *time.Time `json:"updated_at,omitempty"`
}

// Device represents a WhatsApp device/session
type Device struct {
	DeviceID     string     `json:"device_id"`
	APIKeyID     int64      `json:"api_key_id,omitempty"`
	DeviceSecret string     `json:"device_secret,omitempty"` // Only returned on creation
	DeviceName   string     `json:"device_name,omitempty"`
	WhatsMeowJID string     `json:"whatsmeow_jid,omitempty"`
	Status       string     `json:"status"` // pending, active, disconnected, logged_out
	JWTVersion   int        `json:"jwt_version,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	LastActiveAt *time.Time `json:"last_active_at,omitempty"`
}

// DeviceRouting for backward compatibility
type DeviceRouting struct {
	DeviceID     string
	WhatsMeowJID string
	IsActive     bool
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    *time.Time
}

var (
	routingDB   *sql.DB
	routingOnce sync.Once
	routingErr  error

	// JWT Version Cache - prevents DB hit on every request
	jwtVersionCache    = make(map[string]jwtVersionCacheEntry)
	jwtVersionCacheMu  sync.RWMutex
	jwtVersionCacheTTL = 30 * time.Second // Cache JWT version for 30 seconds

	// API Key Cache - prevents DB hit on device creation requests
	apiKeyCache    = make(map[string]apiKeyCacheEntry)
	apiKeyCacheMu  sync.RWMutex
	apiKeyCacheTTL = 5 * time.Minute // Cache API keys for 5 minutes
)

// jwtVersionCacheEntry caches device JWT version to avoid DB queries on every request
type jwtVersionCacheEntry struct {
	version   int
	expiresAt time.Time
}

// apiKeyCacheEntry caches API key data to avoid DB queries
type apiKeyCacheEntry struct {
	apiKey    *APIKey
	expiresAt time.Time
}

func openRoutingDB() (*sql.DB, error) {
	routingOnce.Do(func() {
		driver := datastoreDriver
		dsn := datastoreDSN
		if driver == "" || dsn == "" {
			routingErr = errors.New("whatsapp datastore configuration not initialized")
			return
		}
		if driver != "pgx" {
			routingErr = fmt.Errorf("unsupported datastore driver for routing: %s", driver)
			return
		}
		db, err := sql.Open(driver, dsn)
		if err != nil {
			routingErr = err
			return
		}
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(10)
		db.SetConnMaxLifetime(10 * time.Minute)
		db.SetConnMaxIdleTime(3 * time.Minute)
		if err = db.Ping(); err != nil {
			routingErr = err
			return
		}
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS device_routing (
			device_id TEXT PRIMARY KEY,
			whatsmeow_jid TEXT,
			is_active BOOLEAN DEFAULT FALSE,
			last_login_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
		if err != nil {
			routingErr = err
			return
		}
		_, err = db.Exec(`ALTER TABLE device_routing DROP CONSTRAINT IF EXISTS device_routing_whatsmeow_jid_key`)
		if err != nil {
			routingErr = err
			return
		}
		_, err = db.Exec(`ALTER TABLE device_routing ALTER COLUMN whatsmeow_jid DROP NOT NULL`)
		if err != nil {
			routingErr = err
			return
		}
		_, err = db.Exec(`ALTER TABLE device_routing ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT FALSE`)
		if err != nil {
			routingErr = err
			return
		}
		_, err = db.Exec(`ALTER TABLE device_routing ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMP`)
		if err != nil {
			routingErr = err
			return
		}
		_, err = db.Exec(`ALTER TABLE device_routing ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`)
		if err != nil {
			routingErr = err
			return
		}
		_, err = db.Exec(`ALTER TABLE device_routing ALTER COLUMN is_active SET DEFAULT FALSE`)
		if err != nil {
			routingErr = err
			return
		}
		_, err = db.Exec(`ALTER TABLE device_routing ALTER COLUMN updated_at SET DEFAULT CURRENT_TIMESTAMP`)
		if err != nil {
			routingErr = err
			return
		}
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS wa_webhooks (
			id SERIAL PRIMARY KEY,
			device_id TEXT NOT NULL,
			url TEXT NOT NULL,
			secret TEXT NOT NULL,
			events JSONB NOT NULL DEFAULT '["message.received","connection.connected","connection.disconnected"]'::jsonb,
			active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
		if err != nil {
			routingErr = err
			return
		}
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS wa_webhook_deliveries (
			id BIGSERIAL PRIMARY KEY,
			webhook_id INTEGER NOT NULL REFERENCES wa_webhooks(id) ON DELETE CASCADE,
			event_type TEXT NOT NULL,
			status TEXT NOT NULL,
			attempt_count INTEGER NOT NULL DEFAULT 0,
			last_error TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
		if err != nil {
			routingErr = err
			return
		}
		_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_wa_webhooks_device ON wa_webhooks(device_id)`)
		if err != nil {
			routingErr = err
			return
		}
		_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_wa_webhook_deliveries_webhook ON wa_webhook_deliveries(webhook_id)`)
		if err != nil {
			routingErr = err
			return
		}
		if err := ensureWebhookSchema(db); err != nil {
			routingErr = err
			return
		}
		// Create API keys table for B2B authentication
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS api_keys (
			id SERIAL PRIMARY KEY,
			api_key VARCHAR(64) UNIQUE NOT NULL,
			customer_name VARCHAR(255) NOT NULL,
			customer_email VARCHAR(255) NOT NULL,
			customer_phone VARCHAR(50) NOT NULL,
			max_devices INT DEFAULT 1,
			rate_limit_per_hour INT DEFAULT 1000,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
		if err != nil {
			routingErr = err
			return
		}
		// Migration: Add customer_phone column if it doesn't exist (for existing databases)
		_, _ = db.Exec(`ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS customer_phone VARCHAR(50) NOT NULL DEFAULT ''`)
		if err != nil {
			routingErr = err
			return
		}
		_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_api_keys_key ON api_keys(api_key)`)
		if err != nil {
			routingErr = err
			return
		}
		// Create devices table for device management
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS devices (
			device_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			api_key_id INT REFERENCES api_keys(id) ON DELETE CASCADE,
			device_secret VARCHAR(64) NOT NULL,
			device_name VARCHAR(255),
			whatsmeow_jid VARCHAR(255),
			status VARCHAR(20) DEFAULT 'pending',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_active_at TIMESTAMP
		)`)
		if err != nil {
			routingErr = err
			return
		}
		_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_devices_api_key ON devices(api_key_id)`)
		if err != nil {
			routingErr = err
			return
		}
		_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_devices_secret ON devices(device_id, device_secret)`)
		if err != nil {
			routingErr = err
			return
		}
		// Add jwt_version column for token invalidation (migration)
		_, err = db.Exec(`ALTER TABLE devices ADD COLUMN IF NOT EXISTS jwt_version INT DEFAULT 1`)
		if err != nil {
			routingErr = err
			return
		}

		// Add per-device proxy configuration (migration)
		_, err = db.Exec(`ALTER TABLE devices ADD COLUMN IF NOT EXISTS proxy_url TEXT`)
		if err != nil {
			routingErr = err
			return
		}
		_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_devices_proxy ON devices(device_id) WHERE proxy_url IS NOT NULL`)
		if err != nil {
			routingErr = err
			return
		}

		// Add push notification registration (migration)
		_, err = db.Exec(`ALTER TABLE devices ADD COLUMN IF NOT EXISTS push_notification_platform VARCHAR(20)`)
		if err != nil {
			routingErr = err
			return
		}
		_, err = db.Exec(`ALTER TABLE devices ADD COLUMN IF NOT EXISTS push_notification_token TEXT`)
		if err != nil {
			routingErr = err
			return
		}
		_, err = db.Exec(`ALTER TABLE devices ADD COLUMN IF NOT EXISTS push_notification_registered_at TIMESTAMP`)
		if err != nil {
			routingErr = err
			return
		}
		_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_devices_push ON devices(device_id) WHERE push_notification_platform IS NOT NULL`)
		if err != nil {
			routingErr = err
			return
		}

		// Add passive mode toggle (migration)
		_, err = db.Exec(`ALTER TABLE devices ADD COLUMN IF NOT EXISTS passive_mode BOOLEAN DEFAULT FALSE`)
		if err != nil {
			routingErr = err
			return
		}

		routingDB = db
	})
	return routingDB, routingErr
}

func columnExists(db *sql.DB, table string, column string) (bool, error) {
	var exists bool
	err := db.QueryRow(`SELECT EXISTS (
		SELECT 1 FROM information_schema.columns 
		WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2
	)`, table, column).Scan(&exists)
	return exists, err
}

func columnType(db *sql.DB, table string, column string) (string, error) {
	var dataType string
	err := db.QueryRow(`SELECT data_type FROM information_schema.columns 
		WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2`, table, column).Scan(&dataType)
	if err != nil {
		return "", err
	}
	return dataType, nil
}

func ensureWebhookSchema(db *sql.DB) error {
	if ok, _ := columnExists(db, "wa_webhooks", "is_active"); ok {
		if ok2, _ := columnExists(db, "wa_webhooks", "active"); !ok2 {
			if _, err := db.Exec(`ALTER TABLE wa_webhooks RENAME COLUMN is_active TO active`); err != nil {
				return err
			}
		}
	}
	if ok, _ := columnExists(db, "wa_webhooks", "retry_limit"); ok {
		if _, err := db.Exec(`ALTER TABLE wa_webhooks DROP COLUMN IF EXISTS retry_limit`); err != nil {
			return err
		}
	}
	if ok, _ := columnExists(db, "wa_webhooks", "events"); ok {
		typ, err := columnType(db, "wa_webhooks", "events")
		if err == nil && typ == "ARRAY" {
			if _, err := db.Exec(`ALTER TABLE wa_webhooks ALTER COLUMN events TYPE jsonb USING array_to_json(events)::jsonb`); err != nil {
				return err
			}
		}
	}
	if ok, _ := columnExists(db, "wa_webhooks", "updated_at"); !ok {
		if _, err := db.Exec(`ALTER TABLE wa_webhooks ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`); err != nil {
			return err
		}
	}

	if ok, _ := columnExists(db, "wa_webhook_deliveries", "attempts"); ok {
		if ok2, _ := columnExists(db, "wa_webhook_deliveries", "attempt_count"); !ok2 {
			if _, err := db.Exec(`ALTER TABLE wa_webhook_deliveries RENAME COLUMN attempts TO attempt_count`); err != nil {
				return err
			}
		}
	}
	if ok, _ := columnExists(db, "wa_webhook_deliveries", "error"); ok {
		if ok2, _ := columnExists(db, "wa_webhook_deliveries", "last_error"); !ok2 {
			if _, err := db.Exec(`ALTER TABLE wa_webhook_deliveries RENAME COLUMN error TO last_error`); err != nil {
				return err
			}
		}
	}
	if ok, _ := columnExists(db, "wa_webhook_deliveries", "response_status"); ok {
		if _, err := db.Exec(`ALTER TABLE wa_webhook_deliveries DROP COLUMN IF EXISTS response_status`); err != nil {
			return err
		}
	}
	if ok, _ := columnExists(db, "wa_webhook_deliveries", "device_id"); ok {
		if _, err := db.Exec(`ALTER TABLE wa_webhook_deliveries DROP COLUMN IF EXISTS device_id`); err != nil {
			return err
		}
	}
	if ok, _ := columnExists(db, "wa_webhook_deliveries", "updated_at"); !ok {
		if _, err := db.Exec(`ALTER TABLE wa_webhook_deliveries ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`); err != nil {
			return err
		}
	}
	return nil
}

func SaveDeviceRouting(ctx context.Context, deviceID string, whatsmeowJID string) error {
	db, err := openRoutingDB()
	if err != nil {
		return err
	}
	if whatsmeowJID == "" {
		_, err = db.ExecContext(ctx, `INSERT INTO device_routing (device_id, whatsmeow_jid, is_active, last_login_at, updated_at) VALUES ($1, NULL, FALSE, NULL, NOW()) ON CONFLICT(device_id) DO NOTHING`, deviceID)
	} else {
		_, err = db.ExecContext(ctx, `
			UPDATE device_routing 
			SET whatsmeow_jid = NULL, is_active = FALSE, updated_at = NOW()
			WHERE whatsmeow_jid = $2 AND device_id != $1
		`, deviceID, whatsmeowJID)
		if err != nil {
			return err
		}

		_, err = db.ExecContext(ctx, `
			INSERT INTO device_routing (device_id, whatsmeow_jid, is_active, last_login_at, updated_at) 
			VALUES ($1, $2, TRUE, NOW(), NOW()) 
			ON CONFLICT(device_id) DO UPDATE 
			SET whatsmeow_jid = EXCLUDED.whatsmeow_jid, 
			    is_active = TRUE, 
			    last_login_at = NOW(), 
			    updated_at = NOW()
		`, deviceID, whatsmeowJID)
	}
	return err
}

func GetWhatsMeowJID(ctx context.Context, deviceID string) (string, bool, error) {
	db, err := openRoutingDB()
	if err != nil {
		return "", false, err
	}
	var jid sql.NullString
	var isActive bool
	err = db.QueryRowContext(ctx, `SELECT whatsmeow_jid, is_active FROM device_routing WHERE device_id = $1`, deviceID).Scan(&jid, &isActive)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, errors.New("device routing not found")
	}
	if err != nil {
		return "", false, err
	}
	if !jid.Valid {
		return "", isActive, nil
	}
	return jid.String, isActive, nil
}

func DeleteDeviceRouting(ctx context.Context, deviceID string) error {
	db, err := openRoutingDB()
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		UPDATE device_routing 
		SET whatsmeow_jid = NULL, is_active = FALSE, last_login_at = NULL, updated_at = NOW()
		WHERE device_id = $1
	`, deviceID)
	return err
}

func ListDeviceRoutings(ctx context.Context) ([]DeviceRouting, error) {
	db, err := openRoutingDB()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `SELECT device_id, whatsmeow_jid, is_active, last_login_at, created_at, updated_at FROM device_routing ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var routings []DeviceRouting
	for rows.Next() {
		var r DeviceRouting
		var jid sql.NullString
		var lastLogin sql.NullTime
		var updatedAt sql.NullTime
		if err := rows.Scan(&r.DeviceID, &jid, &r.IsActive, &lastLogin, &r.CreatedAt, &updatedAt); err != nil {
			return nil, err
		}
		if jid.Valid {
			r.WhatsMeowJID = jid.String
		}
		if lastLogin.Valid {
			value := lastLogin.Time
			r.LastLoginAt = &value
		}
		if updatedAt.Valid {
			value := updatedAt.Time
			r.UpdatedAt = &value
		}
		routings = append(routings, r)
	}
	return routings, rows.Err()
}

func GetDeviceIDByJID(ctx context.Context, whatsmeowJID string) (string, error) {
	db, err := openRoutingDB()
	if err != nil {
		return "", err
	}
	var deviceID string
	err = db.QueryRowContext(ctx, `SELECT device_id FROM device_routing WHERE whatsmeow_jid = $1 AND is_active = TRUE`, whatsmeowJID).Scan(&deviceID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.New("device id not found for jid")
	}
	return deviceID, err
}

func SyncDeviceRoutings(ctx context.Context) error {
	db, err := openRoutingDB()
	if err != nil {
		return err
	}

	devices, err := WhatsAppDatastore.GetAllDevices(ctx)
	if err != nil {
		return fmt.Errorf("failed to get whatsmeow devices: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, device := range devices {
		if device.ID == nil {
			continue
		}

		storeJID := device.ID.String()

		var existingDeviceID string
		err := tx.QueryRowContext(ctx, `SELECT device_id FROM device_routing WHERE whatsmeow_jid = $1`, storeJID).Scan(&existingDeviceID)

		if errors.Is(err, sql.ErrNoRows) {
			_, err = tx.ExecContext(ctx, `
				UPDATE device_routing 
				SET whatsmeow_jid = $1, is_active = TRUE, last_login_at = NOW(), updated_at = NOW()
				WHERE whatsmeow_jid IS NULL 
				AND device_id IN (
					SELECT device_id FROM device_routing 
					WHERE whatsmeow_jid IS NULL 
					ORDER BY created_at ASC 
					LIMIT 1
				)
			`, storeJID)
			if err != nil {
				return fmt.Errorf("failed to assign whatsmeow_jid to existing device_id: %w", err)
			}
		} else if err == nil {
			_, err = tx.ExecContext(ctx, `
				UPDATE device_routing 
				SET is_active = TRUE, updated_at = NOW()
				WHERE whatsmeow_jid = $1
			`, storeJID)
			if err != nil {
				return fmt.Errorf("failed to activate device routing: %w", err)
			}
		} else {
			return fmt.Errorf("failed to query device routing: %w", err)
		}
	}

	return tx.Commit()
}

// ============================================================================
// API Key Management Functions
// ============================================================================

// GenerateAPIKey generates a new API key with prefix "wam_"
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 14) // 28 hex chars
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "wam_" + hex.EncodeToString(bytes), nil
}

// GenerateDeviceSecret generates a 64-char hex device secret
func GenerateDeviceSecret() (string, error) {
	bytes := make([]byte, 32) // 64 hex chars
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateAPIKey creates a new API key for a customer
// All parameters are required: customerName, customerEmail, customerPhone, maxDevices (default 1), rateLimitPerHour
func CreateAPIKey(ctx context.Context, customerName, customerEmail, customerPhone string, maxDevices, rateLimitPerHour int) (*APIKey, error) {
	db, err := openRoutingDB()
	if err != nil {
		return nil, err
	}

	apiKey, err := GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Default max_devices to 1 if not specified (but validation should catch this)
	if maxDevices <= 0 {
		maxDevices = 1
	}
	if rateLimitPerHour <= 0 {
		rateLimitPerHour = 1000
	}

	var id int64
	var createdAt time.Time
	err = db.QueryRowContext(ctx, `
		INSERT INTO api_keys (api_key, customer_name, customer_email, customer_phone, max_devices, rate_limit_per_hour)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, apiKey, customerName, customerEmail, customerPhone, maxDevices, rateLimitPerHour).Scan(&id, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	return &APIKey{
		ID:               id,
		APIKey:           apiKey,
		CustomerName:     customerName,
		CustomerEmail:    customerEmail,
		CustomerPhone:    customerPhone,
		MaxDevices:       maxDevices,
		RateLimitPerHour: rateLimitPerHour,
		IsActive:         true,
		CreatedAt:        createdAt,
	}, nil
}

// GetAPIKeyByKey retrieves an API key by its key string (with caching)
func GetAPIKeyByKey(ctx context.Context, apiKey string) (*APIKey, error) {
	// Check cache first (fast path)
	apiKeyCacheMu.RLock()
	if entry, ok := apiKeyCache[apiKey]; ok && time.Now().Before(entry.expiresAt) {
		apiKeyCacheMu.RUnlock()
		return entry.apiKey, nil
	}
	apiKeyCacheMu.RUnlock()

	// Cache miss - fetch from DB
	db, err := openRoutingDB()
	if err != nil {
		return nil, err
	}

	var ak APIKey
	var email, phone sql.NullString
	var updatedAt sql.NullTime
	err = db.QueryRowContext(ctx, `
		SELECT id, api_key, customer_name, customer_email, customer_phone, max_devices, rate_limit_per_hour, is_active, created_at, updated_at
		FROM api_keys WHERE api_key = $1
	`, apiKey).Scan(&ak.ID, &ak.APIKey, &ak.CustomerName, &email, &phone, &ak.MaxDevices, &ak.RateLimitPerHour, &ak.IsActive, &ak.CreatedAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("API key not found")
	}
	if err != nil {
		return nil, err
	}
	if email.Valid {
		ak.CustomerEmail = email.String
	}
	if phone.Valid {
		ak.CustomerPhone = phone.String
	}
	if updatedAt.Valid {
		ak.UpdatedAt = &updatedAt.Time
	}

	// Store in cache
	apiKeyCacheMu.Lock()
	apiKeyCache[apiKey] = apiKeyCacheEntry{
		apiKey:    &ak,
		expiresAt: time.Now().Add(apiKeyCacheTTL),
	}
	apiKeyCacheMu.Unlock()

	return &ak, nil
}

// InvalidateAPIKeyCache removes an API key from cache
// Call this when API key is updated or deleted
func InvalidateAPIKeyCache(apiKey string) {
	apiKeyCacheMu.Lock()
	delete(apiKeyCache, apiKey)
	apiKeyCacheMu.Unlock()
}

// GetAPIKeyByID retrieves an API key by its ID
func GetAPIKeyByID(ctx context.Context, id int64) (*APIKey, error) {
	db, err := openRoutingDB()
	if err != nil {
		return nil, err
	}

	var ak APIKey
	var email, phone sql.NullString
	var updatedAt sql.NullTime
	err = db.QueryRowContext(ctx, `
		SELECT id, api_key, customer_name, customer_email, customer_phone, max_devices, rate_limit_per_hour, is_active, created_at, updated_at
		FROM api_keys WHERE id = $1
	`, id).Scan(&ak.ID, &ak.APIKey, &ak.CustomerName, &email, &phone, &ak.MaxDevices, &ak.RateLimitPerHour, &ak.IsActive, &ak.CreatedAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("API key not found")
	}
	if err != nil {
		return nil, err
	}
	if email.Valid {
		ak.CustomerEmail = email.String
	}
	if phone.Valid {
		ak.CustomerPhone = phone.String
	}
	if updatedAt.Valid {
		ak.UpdatedAt = &updatedAt.Time
	}
	return &ak, nil
}

// ListAPIKeys retrieves all API keys
func ListAPIKeys(ctx context.Context) ([]APIKey, error) {
	db, err := openRoutingDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, `
		SELECT id, api_key, customer_name, customer_email, customer_phone, max_devices, rate_limit_per_hour, is_active, created_at, updated_at
		FROM api_keys ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var ak APIKey
		var email, phone sql.NullString
		var updatedAt sql.NullTime
		if err := rows.Scan(&ak.ID, &ak.APIKey, &ak.CustomerName, &email, &phone, &ak.MaxDevices, &ak.RateLimitPerHour, &ak.IsActive, &ak.CreatedAt, &updatedAt); err != nil {
			return nil, err
		}
		if email.Valid {
			ak.CustomerEmail = email.String
		}
		if phone.Valid {
			ak.CustomerPhone = phone.String
		}
		if updatedAt.Valid {
			ak.UpdatedAt = &updatedAt.Time
		}
		keys = append(keys, ak)
	}
	return keys, rows.Err()
}

// UpdateAPIKey updates an API key
func UpdateAPIKey(ctx context.Context, id int64, customerName, customerEmail, customerPhone string, maxDevices, rateLimitPerHour int, isActive bool) error {
	db, err := openRoutingDB()
	if err != nil {
		return err
	}

	// Get the API key string first to invalidate cache
	var apiKeyStr string
	_ = db.QueryRowContext(ctx, `SELECT api_key FROM api_keys WHERE id = $1`, id).Scan(&apiKeyStr)
	if apiKeyStr != "" {
		InvalidateAPIKeyCache(apiKeyStr)
	}

	_, err = db.ExecContext(ctx, `
		UPDATE api_keys 
		SET customer_name = $2, customer_email = $3, customer_phone = $4, max_devices = $5, rate_limit_per_hour = $6, is_active = $7, updated_at = NOW()
		WHERE id = $1
	`, id, customerName, customerEmail, customerPhone, maxDevices, rateLimitPerHour, isActive)
	return err
}

// DeleteAPIKey deletes an API key and all associated devices
func DeleteAPIKey(ctx context.Context, id int64) error {
	db, err := openRoutingDB()
	if err != nil {
		return err
	}

	// Get the API key string first to invalidate cache
	var apiKeyStr string
	_ = db.QueryRowContext(ctx, `SELECT api_key FROM api_keys WHERE id = $1`, id).Scan(&apiKeyStr)
	if apiKeyStr != "" {
		InvalidateAPIKeyCache(apiKeyStr)
	}

	_, err = db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = $1`, id)
	return err
}

// RegenerateAPIKey generates a new API key for an existing customer
func RegenerateAPIKey(ctx context.Context, id int64) (string, error) {
	db, err := openRoutingDB()
	if err != nil {
		return "", err
	}

	newKey, err := GenerateAPIKey()
	if err != nil {
		return "", err
	}

	result, err := db.ExecContext(ctx, `UPDATE api_keys SET api_key = $2, updated_at = NOW() WHERE id = $1`, id, newKey)
	if err != nil {
		return "", err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return "", errors.New("API key not found")
	}
	return newKey, nil
}

// CountDevicesByAPIKey returns the number of devices for an API key
func CountDevicesByAPIKey(ctx context.Context, apiKeyID int64) (int, error) {
	db, err := openRoutingDB()
	if err != nil {
		return 0, err
	}

	var count int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM devices WHERE api_key_id = $1`, apiKeyID).Scan(&count)
	return count, err
}

// ============================================================================
// Device Management Functions (New B2B System)
// ============================================================================

// CreateDevice creates a new device for an API key
func CreateDevice(ctx context.Context, apiKeyID int64, deviceName string) (*Device, error) {
	db, err := openRoutingDB()
	if err != nil {
		return nil, err
	}

	// Check device limit
	var maxDevices, currentCount int
	err = db.QueryRowContext(ctx, `SELECT max_devices FROM api_keys WHERE id = $1 AND is_active = TRUE`, apiKeyID).Scan(&maxDevices)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("API key not found or inactive")
	}
	if err != nil {
		return nil, err
	}

	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM devices WHERE api_key_id = $1`, apiKeyID).Scan(&currentCount)
	if err != nil {
		return nil, err
	}
	if currentCount >= maxDevices {
		return nil, fmt.Errorf("device limit reached: %d/%d", currentCount, maxDevices)
	}

	secret, err := GenerateDeviceSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate device secret: %w", err)
	}

	var deviceID string
	var createdAt time.Time
	err = db.QueryRowContext(ctx, `
		INSERT INTO devices (api_key_id, device_secret, device_name, status)
		VALUES ($1, $2, $3, 'pending')
		RETURNING device_id, created_at
	`, apiKeyID, secret, deviceName).Scan(&deviceID, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	// Also create entry in device_routing for backward compatibility
	_, err = db.ExecContext(ctx, `
		INSERT INTO device_routing (device_id, whatsmeow_jid, is_active, created_at, updated_at)
		VALUES ($1, NULL, FALSE, NOW(), NOW())
		ON CONFLICT (device_id) DO NOTHING
	`, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to create device routing: %w", err)
	}

	return &Device{
		DeviceID:     deviceID,
		APIKeyID:     apiKeyID,
		DeviceSecret: secret, // Only returned on creation
		DeviceName:   deviceName,
		Status:       "pending",
		JWTVersion:   1, // Initial JWT version
		CreatedAt:    createdAt,
	}, nil
}

// ValidateDeviceCredentials validates device_id and device_secret, returns device with jwt_version
func ValidateDeviceCredentials(ctx context.Context, deviceID, deviceSecret string) (*Device, error) {
	db, err := openRoutingDB()
	if err != nil {
		return nil, err
	}

	var d Device
	var apiKeyID int64
	var name sql.NullString
	var jid sql.NullString
	var lastActive sql.NullTime
	var jwtVersion int

	err = db.QueryRowContext(ctx, `
		SELECT d.device_id, d.api_key_id, d.device_name, d.whatsmeow_jid, d.status, d.jwt_version, d.created_at, d.last_active_at
		FROM devices d
		JOIN api_keys a ON d.api_key_id = a.id
		WHERE d.device_id = $1 AND d.device_secret = $2 AND a.is_active = TRUE
	`, deviceID, deviceSecret).Scan(&d.DeviceID, &apiKeyID, &name, &jid, &d.Status, &jwtVersion, &d.CreatedAt, &lastActive)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("invalid device credentials")
	}
	if err != nil {
		return nil, err
	}

	d.APIKeyID = apiKeyID
	d.JWTVersion = jwtVersion
	if name.Valid {
		d.DeviceName = name.String
	}
	if jid.Valid {
		d.WhatsMeowJID = jid.String
	}
	if lastActive.Valid {
		d.LastActiveAt = &lastActive.Time
	}

	// Update last_active_at
	_, _ = db.ExecContext(ctx, `UPDATE devices SET last_active_at = NOW() WHERE device_id = $1`, deviceID)

	return &d, nil
}

// GetDeviceJWTVersion gets the current jwt_version for a device (with caching)
// This is called on EVERY authenticated request, so caching is critical for performance
func GetDeviceJWTVersion(ctx context.Context, deviceID string) (int, error) {
	// Check cache first (fast path - no DB hit)
	jwtVersionCacheMu.RLock()
	if entry, ok := jwtVersionCache[deviceID]; ok && time.Now().Before(entry.expiresAt) {
		jwtVersionCacheMu.RUnlock()
		return entry.version, nil
	}
	jwtVersionCacheMu.RUnlock()

	// Cache miss - fetch from DB
	db, err := openRoutingDB()
	if err != nil {
		return 0, err
	}

	var version int
	err = db.QueryRowContext(ctx, `SELECT jwt_version FROM devices WHERE device_id = $1`, deviceID).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errors.New("device not found")
	}
	if err != nil {
		return 0, err
	}

	// Store in cache
	jwtVersionCacheMu.Lock()
	jwtVersionCache[deviceID] = jwtVersionCacheEntry{
		version:   version,
		expiresAt: time.Now().Add(jwtVersionCacheTTL),
	}
	jwtVersionCacheMu.Unlock()

	return version, nil
}

// InvalidateJWTVersionCache removes a device from the JWT version cache
// Call this when a token is revoked/regenerated
func InvalidateJWTVersionCache(deviceID string) {
	jwtVersionCacheMu.Lock()
	delete(jwtVersionCache, deviceID)
	jwtVersionCacheMu.Unlock()
}

// IncrementDeviceJWTVersion increments the jwt_version and returns the new version
func IncrementDeviceJWTVersion(ctx context.Context, deviceID string) (int, error) {
	db, err := openRoutingDB()
	if err != nil {
		return 0, err
	}

	var newVersion int
	err = db.QueryRowContext(ctx, `
		UPDATE devices SET jwt_version = jwt_version + 1, last_active_at = NOW()
		WHERE device_id = $1
		RETURNING jwt_version
	`, deviceID).Scan(&newVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errors.New("device not found")
	}
	if err != nil {
		return newVersion, err
	}

	// Invalidate cache so next request fetches new version
	InvalidateJWTVersionCache(deviceID)

	return newVersion, nil
}

// GetDeviceByID retrieves a device by ID (for admin use)
func GetDeviceByID(ctx context.Context, deviceID string) (*Device, error) {
	db, err := openRoutingDB()
	if err != nil {
		return nil, err
	}

	var d Device
	var apiKeyID sql.NullInt64
	var name sql.NullString
	var jid sql.NullString
	var lastActive sql.NullTime

	err = db.QueryRowContext(ctx, `
		SELECT device_id, api_key_id, device_name, whatsmeow_jid, status, created_at, last_active_at
		FROM devices WHERE device_id = $1
	`, deviceID).Scan(&d.DeviceID, &apiKeyID, &name, &jid, &d.Status, &d.CreatedAt, &lastActive)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("device not found")
	}
	if err != nil {
		return nil, err
	}

	if apiKeyID.Valid {
		d.APIKeyID = apiKeyID.Int64
	}
	if name.Valid {
		d.DeviceName = name.String
	}
	if jid.Valid {
		d.WhatsMeowJID = jid.String
	}
	if lastActive.Valid {
		d.LastActiveAt = &lastActive.Time
	}

	return &d, nil
}

// GetDisconnectedDevices retrieves all devices with status "disconnected" that have a valid whatsmeow_jid
// These are devices that should be recovered/reconnected
func GetDisconnectedDevices(ctx context.Context) ([]Device, error) {
	db, err := openRoutingDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, `
		SELECT device_id, api_key_id, device_name, whatsmeow_jid, status, created_at, last_active_at
		FROM devices
		WHERE status = 'disconnected' AND whatsmeow_jid IS NOT NULL AND whatsmeow_jid != ''
		ORDER BY last_active_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var d Device
		var apiKeyID sql.NullInt64
		var name sql.NullString
		var jid sql.NullString
		var lastActive sql.NullTime

		if err := rows.Scan(&d.DeviceID, &apiKeyID, &name, &jid, &d.Status, &d.CreatedAt, &lastActive); err != nil {
			continue
		}

		if apiKeyID.Valid {
			d.APIKeyID = apiKeyID.Int64
		}
		if name.Valid {
			d.DeviceName = name.String
		}
		if jid.Valid {
			d.WhatsMeowJID = jid.String
		}
		if lastActive.Valid {
			d.LastActiveAt = &lastActive.Time
		}

		devices = append(devices, d)
	}

	return devices, nil
}

// GetDevicesNeedingRecovery retrieves all devices that have a valid whatsmeow_jid
// This includes both 'disconnected' AND 'active' status devices that may need restoration after restart
// The difference from GetDisconnectedDevices is that this function doesn't filter by status
func GetDevicesNeedingRecovery(ctx context.Context) ([]Device, error) {
	db, err := openRoutingDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, `
		SELECT device_id, api_key_id, device_name, whatsmeow_jid, status, created_at, last_active_at
		FROM devices
		WHERE whatsmeow_jid IS NOT NULL AND whatsmeow_jid != ''
		AND status IN ('active', 'disconnected')
		ORDER BY last_active_at DESC NULLS LAST
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var d Device
		var apiKeyID sql.NullInt64
		var name sql.NullString
		var jid sql.NullString
		var lastActive sql.NullTime

		if err := rows.Scan(&d.DeviceID, &apiKeyID, &name, &jid, &d.Status, &d.CreatedAt, &lastActive); err != nil {
			continue
		}

		if apiKeyID.Valid {
			d.APIKeyID = apiKeyID.Int64
		}
		if name.Valid {
			d.DeviceName = name.String
		}
		if jid.Valid {
			d.WhatsMeowJID = jid.String
		}
		if lastActive.Valid {
			d.LastActiveAt = &lastActive.Time
		}

		devices = append(devices, d)
	}

	return devices, nil
}

// ListDevicesByAPIKey lists all devices for an API key
func ListDevicesByAPIKey(ctx context.Context, apiKeyID int64) ([]Device, error) {
	db, err := openRoutingDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, `
		SELECT device_id, api_key_id, device_name, whatsmeow_jid, status, created_at, last_active_at
		FROM devices WHERE api_key_id = $1 ORDER BY created_at DESC
	`, apiKeyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var d Device
		var apiKeyID sql.NullInt64
		var name sql.NullString
		var jid sql.NullString
		var lastActive sql.NullTime
		if err := rows.Scan(&d.DeviceID, &apiKeyID, &name, &jid, &d.Status, &d.CreatedAt, &lastActive); err != nil {
			return nil, err
		}
		if apiKeyID.Valid {
			d.APIKeyID = apiKeyID.Int64
		}
		if name.Valid {
			d.DeviceName = name.String
		}
		if jid.Valid {
			d.WhatsMeowJID = jid.String
		}
		if lastActive.Valid {
			d.LastActiveAt = &lastActive.Time
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

// UpdateDeviceStatus updates the device status
func UpdateDeviceStatus(ctx context.Context, deviceID, status string) error {
	db, err := openRoutingDB()
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `UPDATE devices SET status = $2, last_active_at = NOW() WHERE device_id = $1`, deviceID, status)
	return err
}

// UpdateDeviceJID updates the WhatsApp JID for a device
func UpdateDeviceJID(ctx context.Context, deviceID, whatsmeowJID string) error {
	db, err := openRoutingDB()
	if err != nil {
		return err
	}

	status := "active"
	if whatsmeowJID == "" {
		status = "disconnected"
	}

	_, err = db.ExecContext(ctx, `
		UPDATE devices SET whatsmeow_jid = $2, status = $3, last_active_at = NOW() WHERE device_id = $1
	`, deviceID, whatsmeowJID, status)
	return err
}

// DeleteDevice deletes a device
func DeleteDevice(ctx context.Context, deviceID string) error {
	db, err := openRoutingDB()
	if err != nil {
		return err
	}

	// Delete from devices table (device_routing will be cleaned separately if needed)
	_, err = db.ExecContext(ctx, `DELETE FROM devices WHERE device_id = $1`, deviceID)
	if err != nil {
		return err
	}

	// Also clean device_routing
	_, err = db.ExecContext(ctx, `DELETE FROM device_routing WHERE device_id = $1`, deviceID)
	return err
}

// GetJIDByDeviceID retrieves the WhatsApp JID for a device from devices table
func GetJIDByDeviceID(ctx context.Context, deviceID string) (string, error) {
	db, err := openRoutingDB()
	if err != nil {
		return "", err
	}

	var jid sql.NullString
	err = db.QueryRowContext(ctx, `SELECT whatsmeow_jid FROM devices WHERE device_id = $1`, deviceID).Scan(&jid)
	if errors.Is(err, sql.ErrNoRows) {
		// Fallback to device_routing for backward compatibility
		return GetDeviceIDByJIDLegacy(ctx, deviceID)
	}
	if err != nil {
		return "", err
	}
	if !jid.Valid || jid.String == "" {
		return "", errors.New("device not logged in")
	}
	return jid.String, nil
}

// GetDeviceIDByJIDLegacy is a fallback for backward compatibility
func GetDeviceIDByJIDLegacy(ctx context.Context, deviceID string) (string, error) {
	jid, _, err := GetWhatsMeowJID(ctx, deviceID)
	return jid, err
}

// ============================================================================
// Per-Device Proxy Configuration Functions
// ============================================================================

// GetDeviceProxy retrieves the proxy URL for a device
func GetDeviceProxy(ctx context.Context, deviceID string) (string, error) {
	db, err := openRoutingDB()
	if err != nil {
		return "", err
	}
	var proxyURL sql.NullString
	err = db.QueryRowContext(ctx, `SELECT proxy_url FROM devices WHERE device_id = $1`, deviceID).Scan(&proxyURL)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.New("device not found")
	}
	if err != nil {
		return "", err
	}
	if !proxyURL.Valid {
		return "", nil // No proxy set
	}
	return proxyURL.String, nil
}

// SetDeviceProxy sets the proxy URL for a device
func SetDeviceProxy(ctx context.Context, deviceID, proxyURL string) error {
	db, err := openRoutingDB()
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `UPDATE devices SET proxy_url = $2 WHERE device_id = $1`, deviceID, proxyURL)
	return err
}

// ============================================================================
// Admin Dashboard Functions
// ============================================================================

// DeviceWithCustomer represents a device with its customer information
type DeviceWithCustomer struct {
	DeviceID     string     `json:"device_id"`
	DeviceName   string     `json:"device_name"`
	APIKeyID     int64      `json:"api_key_id"`
	CustomerName string     `json:"customer_name"`
	WhatsMeowJID string     `json:"whatsmeow_jid"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	LastActiveAt *time.Time `json:"last_active_at"`
}

// AdminStats represents system-wide statistics for admin dashboard
type AdminStats struct {
	TotalAPIKeys       int `json:"total_api_keys"`
	ActiveAPIKeys      int `json:"active_api_keys"`
	TotalDevices       int `json:"total_devices"`
	ConnectedDevices   int `json:"connected_devices"`
	DisconnectedDevices int `json:"disconnected_devices"`
	PendingDevices     int `json:"pending_devices"`
	LoggedOutDevices   int `json:"logged_out_devices"`
	TotalWebhooks      int `json:"total_webhooks"`
	ActiveWebhooks     int `json:"active_webhooks"`
}

// ListAllDevices retrieves all devices across all API keys with customer info
func ListAllDevices(ctx context.Context) ([]DeviceWithCustomer, error) {
	db, err := openRoutingDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, `
		SELECT d.device_id, d.device_name, d.api_key_id, a.customer_name, d.whatsmeow_jid, d.status, d.created_at, d.last_active_at
		FROM devices d
		LEFT JOIN api_keys a ON d.api_key_id = a.id
		ORDER BY d.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []DeviceWithCustomer
	for rows.Next() {
		var d DeviceWithCustomer
		var name sql.NullString
		var customerName sql.NullString
		var jid sql.NullString
		var lastActive sql.NullTime
		var apiKeyID sql.NullInt64

		if err := rows.Scan(&d.DeviceID, &name, &apiKeyID, &customerName, &jid, &d.Status, &d.CreatedAt, &lastActive); err != nil {
			return nil, err
		}

		if name.Valid {
			d.DeviceName = name.String
		}
		if apiKeyID.Valid {
			d.APIKeyID = apiKeyID.Int64
		}
		if customerName.Valid {
			d.CustomerName = customerName.String
		}
		if jid.Valid {
			d.WhatsMeowJID = jid.String
		}
		if lastActive.Valid {
			d.LastActiveAt = &lastActive.Time
		}

		devices = append(devices, d)
	}
	return devices, rows.Err()
}

// GetAdminStats retrieves system-wide statistics for admin dashboard
func GetAdminStats(ctx context.Context) (*AdminStats, error) {
	db, err := openRoutingDB()
	if err != nil {
		return nil, err
	}

	stats := &AdminStats{}

	// Count API keys
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM api_keys`).Scan(&stats.TotalAPIKeys)
	if err != nil {
		return nil, err
	}

	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM api_keys WHERE is_active = TRUE`).Scan(&stats.ActiveAPIKeys)
	if err != nil {
		return nil, err
	}

	// Count devices by status
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM devices`).Scan(&stats.TotalDevices)
	if err != nil {
		return nil, err
	}

	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM devices WHERE status = 'active'`).Scan(&stats.ConnectedDevices)
	if err != nil {
		return nil, err
	}

	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM devices WHERE status = 'disconnected'`).Scan(&stats.DisconnectedDevices)
	if err != nil {
		return nil, err
	}

	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM devices WHERE status = 'pending'`).Scan(&stats.PendingDevices)
	if err != nil {
		return nil, err
	}

	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM devices WHERE status = 'logged_out'`).Scan(&stats.LoggedOutDevices)
	if err != nil {
		return nil, err
	}

	// Count webhooks
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM wa_webhooks`).Scan(&stats.TotalWebhooks)
	if err != nil {
		// Table might not exist, set to 0
		stats.TotalWebhooks = 0
	}

	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM wa_webhooks WHERE active = TRUE`).Scan(&stats.ActiveWebhooks)
	if err != nil {
		stats.ActiveWebhooks = 0
	}

	return stats, nil
}

// GetWebhookStats retrieves webhook delivery statistics
func GetWebhookStats(ctx context.Context) (map[string]interface{}, error) {
	db, err := openRoutingDB()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]interface{})

	var totalWebhooks, activeWebhooks int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM wa_webhooks`).Scan(&totalWebhooks)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM wa_webhooks WHERE active = TRUE`).Scan(&activeWebhooks)

	var totalDeliveries, successDeliveries, failedDeliveries int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM wa_webhook_deliveries`).Scan(&totalDeliveries)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM wa_webhook_deliveries WHERE status = 'success'`).Scan(&successDeliveries)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM wa_webhook_deliveries WHERE status = 'failed'`).Scan(&failedDeliveries)

	var successRate float64
	if totalDeliveries > 0 {
		successRate = float64(successDeliveries) / float64(totalDeliveries) * 100
	}

	stats["total_webhooks"] = totalWebhooks
	stats["active_webhooks"] = activeWebhooks
	stats["total_deliveries"] = totalDeliveries
	stats["success_deliveries"] = successDeliveries
	stats["failed_deliveries"] = failedDeliveries
	stats["success_rate"] = successRate

	return stats, nil
}
