package whatsapp

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	mathrand "math/rand/v2"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/forPelevin/gomoji"
	"github.com/rivo/uniseg"
	"github.com/sunshineplan/imgconv"

	qrCode "github.com/skip2/go-qrcode"
	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"golang.org/x/sync/singleflight"
	"golang.org/x/time/rate"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/webhook"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/env"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/log"
)

// SessionKey uses DeviceID only since it's always known and unique
// JID may be empty initially and only becomes known after QR scan
type SessionKey struct {
	DeviceID string
}

var WhatsAppDatastore *sqlstore.Container

var requiredDatastoreTables = []string{
	"device_routing",
	"whatsmeow_device",
	"whatsmeow_identity_keys",
	"whatsmeow_pre_keys",
	"whatsmeow_sessions",
	"whatsmeow_sender_keys",
	"whatsmeow_app_state_sync_keys",
	"whatsmeow_app_state_version",
	"whatsmeow_app_state_mutation_macs",
	"whatsmeow_contacts",
	"whatsmeow_chat_settings",
	"whatsmeow_message_secrets",
	"whatsmeow_privacy_tokens",
	"whatsmeow_lid_map",
	"whatsmeow_event_buffer",
	"whatsmeow_version",
}

var (
	clientsMu                sync.RWMutex
	WhatsAppClient           = make(map[SessionKey]*whatsmeow.Client)
	WhatsAppClientProxyURL   string
	ErrInvalidGroupID        = errors.New("WhatsApp Group ID is Not Group Server")
	ErrParticipantMustBeUser = errors.New("WhatsApp Participant ID must be a Personal JID")
	datastoreDriver          string
	datastoreDSN             string
	keysDatastoreDriver      string
	keysDatastoreDSN         string
	WhatsAppKeysDatastore    *sqlstore.Container
	webhookEngine            *webhook.Engine
	groupListCacheMu         sync.RWMutex
	groupListCache           = make(map[groupListCacheKey]groupListCacheEntry)
	groupListCacheTTL        = 5 * time.Minute // Extended TTL for multi-device efficiency
	groupListCacheEnabled    = true
	deviceOSName             = "Chrome"
	whatsAppLogLevel         = "ERROR"
	autoMarkReadEnabled      bool
	autoDownloadMediaEnabled bool
	autoReplyEnabled         bool
	autoPresenceEnabled      bool
	autoTypingEnabled        bool
	typingDelayMin           time.Duration
	typingDelayMax           time.Duration
	readReceiptJitterEnabled bool
	readReceiptDelayMin      time.Duration
	readReceiptDelayMax      time.Duration
	appStateWebhookEnabled   bool

	rateLimitEnabled    bool
	rateLimitPerMinute  int
	rateLimitBurstSize  int
	deviceRateLimiters  = make(map[string]*rate.Limiter)
	deviceRateLimiterMu sync.Mutex

	phonePattern = regexp.MustCompile(`^[1-9][0-9]{5,15}$`)

	// Request deduplication: prevents multiple simultaneous requests for the same device
	// from all hitting WhatsApp servers (singleflight pattern)
	groupListInflight   = make(map[string]*inflightRequest)
	groupListInflightMu sync.Mutex

	// isOn cache & batching controls
	isOnCacheEnabled bool
	isOnCacheTTL     time.Duration
	isOnCacheMax     int
	isOnBatchSize    int
	isOnCache        = make(map[string]isOnCacheEntry)
	isOnCacheMu      sync.RWMutex
	isOnSingleFlight singleflight.Group
)

// inflightRequest tracks an ongoing group list fetch to deduplicate concurrent requests
type inflightRequest struct {
	done   chan struct{}
	result []EnhancedGroupInfo
	err    error
}

type groupListCacheKey struct {
	DeviceID      string
	ResolvePhones bool
}

type groupListCacheEntry struct {
	data      []EnhancedGroupInfo
	expiresAt time.Time
}

type isOnCacheEntry struct {
	jid     types.JID
	ok      bool
	expires time.Time
}

const (
	qrChannelWaitTimeout    = 2 * time.Minute
	pairPhoneRequestTimeout = 90 * time.Second
	logoutRequestTimeout    = 30 * time.Second
	routingCleanupTimeout   = 5 * time.Second
	groupFetchTimeout       = 60 * time.Second // 60s for accounts with many groups (290+)
	groupConversionWorkers  = 20               // Number of parallel workers for group conversion
	maxImageBytes           = int64(20 * 1024 * 1024)
	maxDocumentBytes        = int64(50 * 1024 * 1024)
	maxVideoBytes           = int64(100 * 1024 * 1024)
	maxAudioBytes           = int64(20 * 1024 * 1024)
)

func init() {
	var err error

	dbType, err := env.GetEnvString("WHATSAPP_DATASTORE_TYPE")
	if err != nil {
		log.SysErr("db-type", err)
		os.Exit(1)
	}

	dbURI, err := env.GetEnvString("WHATSAPP_DATASTORE_URI")
	if err != nil {
		log.SysErr("db-uri", err)
		os.Exit(1)
	}

	normalizedDriver := normalizeDatastoreDriver(dbType)
	dbURI = normalizeDatastoreDSN(normalizedDriver, dbURI)

	datastoreDriver = normalizedDriver
	datastoreDSN = dbURI

	log.Sys("init-db", normalizedDriver)

	datastore, err := sqlstore.New(context.Background(), normalizedDriver, dbURI, nil)
	if err != nil {
		log.SysErr("db-init", err)
		os.Exit(1)
	}

	// Optional separate datastore for encryption keys and sessions
	if keysURI := strings.TrimSpace(os.Getenv("WHATSAPP_KEYS_DATASTORE_URI")); keysURI != "" {
		keysDatastoreDriver = normalizedDriver
		keysDatastoreDSN = normalizeDatastoreDSN(keysDatastoreDriver, keysURI)
		log.Sys("init-keys-db", keysDatastoreDriver)
		keysStore, keysErr := sqlstore.New(context.Background(), keysDatastoreDriver, keysDatastoreDSN, nil)
		if keysErr != nil {
			log.SysErr("keys-db-init", keysErr)
		} else {
			WhatsAppKeysDatastore = keysStore
		}
	}

	WhatsAppClientProxyURL, _ = env.GetEnvString("WHATSAPP_CLIENT_PROXY_URL")

	if _, err := openRoutingDB(); err != nil {
		log.SysErr("routing-db", err)
		os.Exit(1)
	}

	WhatsAppDatastore = datastore

	if err := upgradeDatastoreSchema(context.Background()); err != nil {
		log.SysErr("db-schema", err)
		os.Exit(1)
	}

	db, err := openRoutingDB()
	if err != nil {
		log.SysErr("webhook-db", err)
		os.Exit(1)
	}
	webhookStore := webhook.NewStore(db)
	webhookEngine = webhook.NewEngine(webhookStore)

	log.Sys("db-ready")

	configureGroupListCache()
	loadBehaviorConfig()
	loadIsOnConfig()
	loadRateLimitConfig()
}

func configureGroupListCache() {
	if ttlRaw, ok := os.LookupEnv("WHATSAPP_GROUP_LIST_CACHE_TTL"); ok {
		ttlRaw = strings.TrimSpace(ttlRaw)
		if ttlRaw != "" {
			if ttl, err := time.ParseDuration(ttlRaw); err == nil && ttl > 0 {
				groupListCacheTTL = ttl
			}
		}
	} else {
		groupListCacheTTL = 5 * time.Minute
	}

	if disabledRaw, ok := os.LookupEnv("WHATSAPP_GROUP_LIST_CACHE_DISABLED"); ok {
		disabledRaw = strings.TrimSpace(disabledRaw)
		if disabledRaw != "" {
			if disabled, err := strconv.ParseBool(disabledRaw); err == nil {
				groupListCacheEnabled = !disabled
			}
		}
	}

	if groupListCacheEnabled {
		log.Sys("cache", fmt.Sprintf("grp-list ttl=%s", groupListCacheTTL))
	}
}

func parseOptionalBool(envKey string, defaultVal bool) bool {
	if raw, ok := os.LookupEnv(envKey); ok {
		raw = strings.TrimSpace(raw)
		if parsed, err := strconv.ParseBool(raw); err == nil {
			return parsed
		}
	}
	return defaultVal
}

func errInvalidBoolValue(raw, key string) error {
	return fmt.Errorf("invalid boolean value %q for %s", raw, key)
}

func parseOptionalDuration(envKey string, defaultVal time.Duration) time.Duration {
	if raw, ok := os.LookupEnv(envKey); ok {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return defaultVal
		}
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			return d
		}
	}
	return defaultVal
}

func parseOptionalInt(envKey string, defaultVal int, minVal int) int {
	if raw, ok := os.LookupEnv(envKey); ok {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return defaultVal
		}
		if val, err := strconv.Atoi(raw); err == nil && val >= minVal {
			return val
		}
	}
	return defaultVal
}

func loadBehaviorConfig() {
	if val := strings.TrimSpace(os.Getenv("WHATSAPP_DEVICE_OS_NAME")); val != "" {
		deviceOSName = val
	}
	if val := strings.TrimSpace(os.Getenv("WHATSAPP_LOG_LEVEL")); val != "" {
		whatsAppLogLevel = strings.ToUpper(val)
	}

	autoMarkReadEnabled = parseOptionalBool("WHATSAPP_AUTO_MARK_READ", false)
	autoDownloadMediaEnabled = parseOptionalBool("WHATSAPP_AUTO_DOWNLOAD_MEDIA", false)
	autoReplyEnabled = parseOptionalBool("WHATSAPP_AUTO_REPLY_ENABLED", false)
	autoPresenceEnabled = parseOptionalBool("WHATSAPP_AUTO_PRESENCE_ENABLED", true)
	autoTypingEnabled = parseOptionalBool("WHATSAPP_AUTO_TYPING_ENABLED", true)
	typingDelayMin = parseOptionalDuration("WHATSAPP_TYPING_DELAY_MIN", 1*time.Second)
	typingDelayMax = parseOptionalDuration("WHATSAPP_TYPING_DELAY_MAX", 3*time.Second)
	if typingDelayMax < typingDelayMin {
		typingDelayMax = typingDelayMin
	}

	readReceiptJitterEnabled = parseOptionalBool("WHATSAPP_READ_RECEIPT_JITTER_ENABLED", true)
	readReceiptDelayMin = parseOptionalDuration("WHATSAPP_READ_RECEIPT_DELAY_MIN", 500*time.Millisecond)
	readReceiptDelayMax = parseOptionalDuration("WHATSAPP_READ_RECEIPT_DELAY_MAX", 2*time.Second)
	if readReceiptDelayMax < readReceiptDelayMin {
		readReceiptDelayMax = readReceiptDelayMin
	}

	appStateWebhookEnabled = parseOptionalBool("WHATSAPP_APPSTATE_WEBHOOK_ENABLED", false)

	log.Sys("cfg", fmt.Sprintf("presence:%t typing:%t read_jitter:%t", autoPresenceEnabled, autoTypingEnabled, readReceiptJitterEnabled))
}

func loadIsOnConfig() {
	isOnCacheEnabled = parseOptionalBool("WHATSAPP_ISON_CACHE_ENABLED", false)

	if ttlRaw, ok := os.LookupEnv("WHATSAPP_ISON_CACHE_TTL"); ok {
		if ttl, err := time.ParseDuration(strings.TrimSpace(ttlRaw)); err == nil && ttl > 0 {
			isOnCacheTTL = ttl
		}
	}
	if isOnCacheTTL == 0 {
		isOnCacheTTL = 5 * time.Minute
	}

	if maxRaw, ok := os.LookupEnv("WHATSAPP_ISON_CACHE_MAX"); ok {
		if maxVal, err := strconv.Atoi(strings.TrimSpace(maxRaw)); err == nil && maxVal > 0 {
			isOnCacheMax = maxVal
		}
	}
	if isOnCacheMax == 0 {
		isOnCacheMax = 1000
	}

	if batchRaw, ok := os.LookupEnv("WHATSAPP_ISON_BATCH_SIZE"); ok {
		if b, err := strconv.Atoi(strings.TrimSpace(batchRaw)); err == nil && b > 0 {
			isOnBatchSize = b
		}
	}
	if isOnBatchSize == 0 {
		isOnBatchSize = 50
	}

	if isOnCacheEnabled {
		log.Sys("cache", fmt.Sprintf("ison ttl=%s", isOnCacheTTL))
	}
}

func loadRateLimitConfig() {
	rateLimitEnabled = parseOptionalBool("WHATSAPP_RATE_LIMIT_ENABLED", false)
	rateLimitPerMinute = parseOptionalInt("WHATSAPP_RATE_LIMIT_MSG_PER_MINUTE", 20, 1)
	rateLimitBurstSize = parseOptionalInt("WHATSAPP_RATE_LIMIT_BURST_SIZE", 5, 1)
	if rateLimitBurstSize > rateLimitPerMinute {
		rateLimitBurstSize = rateLimitPerMinute
	}
	if rateLimitEnabled {
		log.Sys("ratelimit", fmt.Sprintf("%d/min burst:%d", rateLimitPerMinute, rateLimitBurstSize))
	}
}

func jitterDuration(min, max time.Duration) time.Duration {
	if max <= min {
		return min
	}
	delta := max - min
	return min + time.Duration(mathrand.Int64N(int64(delta)+1))
}

type SendOptions struct {
	TypingSimulation   *bool
	PresenceSimulation *bool
}

func rateLimiterForDevice(deviceID string) *rate.Limiter {
	deviceRateLimiterMu.Lock()
	defer deviceRateLimiterMu.Unlock()

	if limiter, ok := deviceRateLimiters[deviceID]; ok {
		return limiter
	}
	limit := rate.Every(time.Minute / time.Duration(rateLimitPerMinute))
	limiter := rate.NewLimiter(limit, rateLimitBurstSize)
	deviceRateLimiters[deviceID] = limiter
	return limiter
}

func waitRateLimit(ctx context.Context, deviceID string) error {
	if !rateLimitEnabled {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return rateLimiterForDevice(deviceID).Wait(ctx)
}

func beginPresenceSimulation(ctx context.Context, jid string, deviceID string, remoteJID types.JID, isAudio bool, opts *SendOptions) func() {
	presenceEnabled := autoPresenceEnabled
	typingEnabled := autoTypingEnabled
	if opts != nil && opts.PresenceSimulation != nil {
		presenceEnabled = *opts.PresenceSimulation
	}
	if opts != nil && opts.TypingSimulation != nil {
		typingEnabled = *opts.TypingSimulation
	}

	if !presenceEnabled {
		return func() {}
	}
	if ctx == nil {
		ctx = context.Background()
	}

	// Go online and start composing (typing/recording)
	WhatsAppPresence(ctx, jid, deviceID, true)
	if typingEnabled {
		WhatsAppComposeStatus(ctx, jid, deviceID, remoteJID, true, isAudio)
		delay := jitterDuration(typingDelayMin, typingDelayMax)
		time.Sleep(delay)
	}

	return func() {
		if typingEnabled {
			WhatsAppComposeStatus(context.Background(), jid, deviceID, remoteJID, false, isAudio)
		}
		WhatsAppPresence(context.Background(), jid, deviceID, false)
	}
}

func loadGroupListCache(deviceID string, resolvePhones bool) ([]EnhancedGroupInfo, bool) {
	if !groupListCacheEnabled || groupListCacheTTL <= 0 {
		return nil, false
	}
	key := groupListCacheKey{DeviceID: deviceID, ResolvePhones: resolvePhones}
	groupListCacheMu.RLock()
	entry, ok := groupListCache[key]
	groupListCacheMu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		groupListCacheMu.Lock()
		delete(groupListCache, key)
		groupListCacheMu.Unlock()
		return nil, false
	}
	return entry.data, true
}

func storeGroupListCache(deviceID string, resolvePhones bool, data []EnhancedGroupInfo) {
	if !groupListCacheEnabled || groupListCacheTTL <= 0 {
		return
	}
	key := groupListCacheKey{DeviceID: deviceID, ResolvePhones: resolvePhones}
	groupListCacheMu.Lock()
	groupListCache[key] = groupListCacheEntry{
		data:      data,
		expiresAt: time.Now().Add(groupListCacheTTL),
	}
	groupListCacheMu.Unlock()
}

func invalidateGroupListCache(deviceID string) {
	if !groupListCacheEnabled {
		return
	}
	groupListCacheMu.Lock()
	delete(groupListCache, groupListCacheKey{DeviceID: deviceID, ResolvePhones: true})
	delete(groupListCache, groupListCacheKey{DeviceID: deviceID, ResolvePhones: false})
	groupListCacheMu.Unlock()
}

// cleanupExpiredCache removes expired entries from the group list cache
// This runs periodically to prevent memory growth with many devices
func cleanupExpiredCache() {
	groupListCacheMu.Lock()
	defer groupListCacheMu.Unlock()

	now := time.Now()
	for key, entry := range groupListCache {
		if now.After(entry.expiresAt) {
			delete(groupListCache, key)
		}
	}
}

// cleanupExpiredJWTVersionCache removes expired JWT version cache entries
func cleanupExpiredJWTVersionCache() {
	jwtVersionCacheMu.Lock()
	defer jwtVersionCacheMu.Unlock()

	now := time.Now()
	for key, entry := range jwtVersionCache {
		if now.After(entry.expiresAt) {
			delete(jwtVersionCache, key)
		}
	}
}

// cleanupExpiredAPIKeyCache removes expired API key cache entries
func cleanupExpiredAPIKeyCache() {
	apiKeyCacheMu.Lock()
	defer apiKeyCacheMu.Unlock()

	now := time.Now()
	for key, entry := range apiKeyCache {
		if now.After(entry.expiresAt) {
			delete(apiKeyCache, key)
		}
	}
}

// StartCacheCleanup starts a background goroutine that periodically cleans expired cache entries
// Call this once during application startup
func StartCacheCleanup() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute) // Run every 10 minutes
		defer ticker.Stop()

		for range ticker.C {
			cleanupExpiredCache()
			cleanupExpiredJWTVersionCache()
			cleanupExpiredAPIKeyCache()
		}
	}()
}

func upgradeDatastoreSchema(ctx context.Context) error {
	if WhatsAppDatastore == nil {
		return errors.New("whatsapp datastore not initialized")
	}

	if err := WhatsAppDatastore.Upgrade(ctx); err != nil {
		return fmt.Errorf("upgrade operation failed: %w", err)
	}

	if WhatsAppKeysDatastore != nil {
		if err := WhatsAppKeysDatastore.Upgrade(ctx); err != nil {
			return fmt.Errorf("keys datastore upgrade operation failed: %w", err)
		}
	}

	return nil
}

func tableExistsQuery(driver string) (string, error) {
	switch strings.ToLower(driver) {
	case "postgres", "pgx":
		return "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = $1)", nil
	default:
		return "", fmt.Errorf("unsupported datastore driver %s", driver)
	}
}

func normalizeDatastoreDriver(driver string) string {
	switch strings.ToLower(driver) {
	case "postgresql", "postgres", "pgx":
		return "pgx"
	default:
		return strings.ToLower(driver)
	}
}

func normalizeDatastoreDSN(driver string, dsn string) string {
	if driver != "pgx" {
		return dsn
	}
	appendParam := func(current string, key string, value string) string {
		if strings.Contains(current, key+"=") {
			return current
		}
		separator := "?"
		if strings.Contains(current, "?") {
			if strings.HasSuffix(current, "?") || strings.HasSuffix(current, "&") {
				separator = ""
			} else {
				separator = "&"
			}
		}
		return current + separator + key + "=" + value
	}
	dsn = appendParam(dsn, "prefer_simple_protocol", "true")
	dsn = appendParam(dsn, "statement_cache_capacity", "0")
	dsn = appendParam(dsn, "default_query_exec_mode", "simple_protocol")
	return dsn
}

// clientKey now uses deviceID only - JID parameter kept for compatibility but ignored
func clientKey(jid string, deviceID string) SessionKey {
	return SessionKey{DeviceID: deviceID}
}

// getClientByDeviceID looks up client by deviceID only (preferred method)
func getClientByDeviceID(deviceID string) *whatsmeow.Client {
	key := SessionKey{DeviceID: deviceID}
	clientsMu.RLock()
	client := WhatsAppClient[key]
	clientsMu.RUnlock()
	return client
}

func getClient(jid string, deviceID string) *whatsmeow.Client {
	return getClientByDeviceID(deviceID)
}

func setClient(jid string, deviceID string, client *whatsmeow.Client) {
	key := SessionKey{DeviceID: deviceID}
	clientsMu.Lock()
	WhatsAppClient[key] = client
	clientsMu.Unlock()

	// Attach keys datastore if configured and device JID is already known
	if client != nil {
		attachKeysStore(client)
	}
}

func deleteClient(jid string, deviceID string) {
	key := SessionKey{DeviceID: deviceID}
	clientsMu.Lock()
	delete(WhatsAppClient, key)
	clientsMu.Unlock()
}

func rangeClients(fn func(SessionKey, *whatsmeow.Client)) {
	clientsMu.RLock()
	keys := make([]SessionKey, 0, len(WhatsAppClient))
	for key := range WhatsAppClient {
		keys = append(keys, key)
	}
	clientsMu.RUnlock()
	for _, key := range keys {
		client := getClientByDeviceID(key.DeviceID)
		if client != nil {
			fn(key, client)
		}
	}
}

func clientsLen() int {
	clientsMu.RLock()
	defer clientsMu.RUnlock()
	return len(WhatsAppClient)
}

// attachKeysStore rebinds the key-related stores to the optional keys datastore.
// This reduces contention on the primary datastore if a separate DB is configured.
func attachKeysStore(client *whatsmeow.Client) {
	if WhatsAppKeysDatastore == nil || client == nil || client.Store == nil || client.Store.ID == nil {
		return
	}
	keysStore := sqlstore.NewSQLStore(WhatsAppKeysDatastore, *client.Store.ID)
	// Only rebind key-heavy stores; keep contacts and chat settings on the primary datastore.
	client.Store.Identities = keysStore
	client.Store.Sessions = keysStore
	client.Store.PreKeys = keysStore
	client.Store.SenderKeys = keysStore
	client.Store.AppStateKeys = keysStore
	client.Store.MsgSecrets = keysStore
	client.Store.EventBuffer = keysStore
}

func WhatsAppClientsLen() int {
	return clientsLen()
}

// maskJIDForLog masks a JID/phone number for secure logging
// Shows only first 3 and last 2 digits: 628123456789 -> 628*****89
func maskJIDForLog(jid string) string {
	if len(jid) < 6 {
		return "***"
	}
	// Strip @suffix if present
	atIdx := len(jid)
	for i, c := range jid {
		if c == '@' {
			atIdx = i
			break
		}
	}
	numPart := jid[:atIdx]
	suffix := jid[atIdx:]

	if len(numPart) < 6 {
		return "***" + suffix
	}
	masked := numPart[:3] + strings.Repeat("*", len(numPart)-5) + numPart[len(numPart)-2:]
	return masked + suffix
}

func WhatsAppRangeClients(fn func(jid string, deviceID string, client *whatsmeow.Client)) {
	rangeClients(func(key SessionKey, client *whatsmeow.Client) {
		// Get JID from client store since it's not in the key anymore
		jid := ""
		if client.Store.ID != nil {
			jid = WhatsAppDecomposeJID(client.Store.ID.User)
		}
		fn(jid, key.DeviceID, client)
	})
}

func currentClient(jid string, deviceID string) (*whatsmeow.Client, error) {
	client := getClient(jid, deviceID)
	if client == nil {
		return nil, errors.New("WhatsApp Client is not Valid")
	}
	return client, nil
}

func WhatsAppInitClient(device *store.Device, jid string, deviceID string) {
	var err error

	existingClient := getClient(jid, deviceID)
	if existingClient != nil {
		return
	}

	log.Evt("init", "client", deviceID)

	if device == nil {
		device = WhatsAppDatastore.NewDevice()
	}

	osName := deviceOSName
	if osName == "" {
		osName = runtime.GOOS
	}
	store.DeviceProps.Os = proto.String(osName)
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_CHROME.Enum()
	store.DeviceProps.RequireFullSync = proto.Bool(false)

	version.Major, err = env.GetEnvInt("WHATSAPP_VERSION_MAJOR")
	if err == nil {
		store.DeviceProps.Version.Primary = proto.Uint32(uint32(version.Major))
	}
	version.Minor, err = env.GetEnvInt("WHATSAPP_VERSION_MINOR")
	if err == nil {
		store.DeviceProps.Version.Secondary = proto.Uint32(uint32(version.Minor))
	}
	version.Patch, err = env.GetEnvInt("WHATSAPP_VERSION_PATCH")
	if err == nil {
		store.DeviceProps.Version.Tertiary = proto.Uint32(uint32(version.Patch))
	}

	baseLogger := waLog.Stdout("Client", whatsAppLogLevel, true)
	client := whatsmeow.NewClient(device, newFilteredLogger(baseLogger))

	if len(WhatsAppClientProxyURL) > 0 {
		client.SetProxyAddress(WhatsAppClientProxyURL)
	}

	client.EnableAutoReconnect = true
	client.AutoTrustIdentity = true

	client.AddEventHandler(handleWhatsAppEvents(jid, deviceID))

	setClient(jid, deviceID, client)

	if device.ID != nil {
		_ = SaveDeviceRouting(context.Background(), deviceID, device.ID.String())
	}
}

// getClientJID retrieves the current JID from the client store, falling back to initialJid if not available
func getClientJID(initialJid string, deviceID string) string {
	client := getClient(initialJid, deviceID)
	if client != nil && client.Store.ID != nil {
		return WhatsAppDecomposeJID(client.Store.ID.User)
	}
	// Try to find client with empty initial JID (new device case)
	if initialJid == "" {
		client = getClient("", deviceID)
		if client != nil && client.Store.ID != nil {
			return WhatsAppDecomposeJID(client.Store.ID.User)
		}
	}
	return initialJid
}

func handleWhatsAppEvents(jid string, deviceID string) func(interface{}) {
	return func(evt interface{}) {
		// Get the current JID dynamically from client store
		currentJID := getClientJID(jid, deviceID)

		switch e := evt.(type) {
		case *events.LoggedOut:
			client, err := currentClient(jid, deviceID)
			if err == nil {
				client.Disconnect()
			}
			deleteClient(jid, deviceID)
			routingCtx, routingCancel := context.WithTimeout(context.Background(), routingCleanupTimeout)
			_ = DeleteDeviceRouting(routingCtx, deviceID)
			// Update device status to logged_out
			_ = UpdateDeviceStatus(routingCtx, deviceID, "logged_out")
			routingCancel()
			dispatchWebhook(deviceID, webhook.EventConnectionLoggedOut, map[string]interface{}{
				"jid": currentJID,
			})
			cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), 10*time.Second)
			purgeDeviceSession(cleanupCtx, deviceID)
			cancelCleanup()
			WhatsAppInitClient(nil, "", deviceID)
		case *events.StreamReplaced:
			client, err := currentClient(jid, deviceID)
			if err == nil {
				client.Disconnect()
			}
			deleteClient(jid, deviceID)
			routingCtx, routingCancel := context.WithTimeout(context.Background(), routingCleanupTimeout)
			_ = DeleteDeviceRouting(routingCtx, deviceID)
			routingCancel()
			dispatchWebhook(deviceID, webhook.EventConnectionReconnecting, map[string]interface{}{
				"jid": currentJID,
			})
		case *events.Connected:
			// Get the JID from client store after connection
			client, err := currentClient(jid, deviceID)
			connectedJID := currentJID
			if err == nil && client.Store.ID != nil {
				connectedJID = WhatsAppDecomposeJID(client.Store.ID.User)
				_ = SaveDeviceRouting(context.Background(), deviceID, client.Store.ID.String())
				_ = UpdateDeviceJID(context.Background(), deviceID, client.Store.ID.String())
				attachKeysStore(client)
			}
			isLoggedIn := false
			isConnected := false
			pushName := ""
			platform := ""
			businessName := ""
			if client != nil && client.Store != nil {
				isLoggedIn = client.IsLoggedIn()
				isConnected = client.IsConnected()
				pushName = client.Store.PushName
				platform = client.Store.Platform
				businessName = client.Store.BusinessName
			}
			log.Conn("connected", deviceID, connectedJID)
			sendAvailablePresence(connectedJID, deviceID)
			dispatchWebhook(deviceID, webhook.EventConnectionConnected, map[string]interface{}{
				"jid":           connectedJID,
				"phone_number":  connectedJID,
				"push_name":     pushName,
				"platform":      platform,
				"business_name": businessName,
				"is_logged_in":  isLoggedIn,
				"is_connected":  isConnected,
			})
		case *events.Disconnected:
			log.Conn("disconnected", deviceID, currentJID)
			_ = UpdateDeviceStatus(context.Background(), deviceID, "disconnected")
			dispatchWebhook(deviceID, webhook.EventConnectionDisconnected, map[string]interface{}{
				"jid": currentJID,
			})
		case *events.KeepAliveTimeout:
			log.Conn("keepalive-timeout", deviceID, currentJID)
			dispatchWebhook(deviceID, webhook.EventConnectionKeepAliveTimeout, map[string]interface{}{
				"jid":          currentJID,
				"error_count":  e.ErrorCount,
				"last_success": e.LastSuccess,
			})
		case *events.TemporaryBan:
			log.Conn("temp-banned", deviceID, currentJID)
			dispatchWebhook(deviceID, webhook.EventConnectionTemporaryBan, map[string]interface{}{
				"jid":     currentJID,
				"reason":  e.Code,
				"expires": e.Expire,
			})
		case *events.Message:
			autoMarkMessageAsRead(currentJID, deviceID, e)
			// Check if this is a message deletion (revoke)
			if e.Message != nil && e.Message.ProtocolMessage != nil &&
				e.Message.ProtocolMessage.GetType() == waE2E.ProtocolMessage_REVOKE {
				// This is a deleted message
				deletedMsgID := e.Message.ProtocolMessage.GetKey().GetID()
				dispatchWebhook(deviceID, webhook.EventMessageDeleted, map[string]interface{}{
					"message_id": deletedMsgID,
					"from":       e.Info.Sender.String(),
					"chat":       e.Info.Chat.String(),
					"timestamp":  e.Info.Timestamp.Unix(),
					"deleted_by": e.Info.Sender.String(),
					"is_from_me": e.Info.IsFromMe,
				})
			} else {
				// Regular message received
				dispatchWebhook(deviceID, webhook.EventMessageReceived, map[string]interface{}{
					"message_id": e.Info.ID,
					"from":       e.Info.Sender.String(),
					"chat":       e.Info.Chat.String(),
					"timestamp":  e.Info.Timestamp.Unix(),
					"is_from_me": e.Info.IsFromMe,
				})
			}
		case *events.Receipt:
			eventType := webhook.EventMessageDelivered
			if e.Type == events.ReceiptTypeRead || e.Type == events.ReceiptTypeReadSelf {
				eventType = webhook.EventMessageRead
			} else if e.Type == events.ReceiptTypePlayed {
				eventType = webhook.EventMessagePlayed
			}
			for _, msgID := range e.MessageIDs {
				dispatchWebhook(deviceID, eventType, map[string]interface{}{
					"message_id": msgID,
					"chat":       e.Chat.String(),
					"sender":     e.Sender.String(),
					"timestamp":  e.Timestamp.Unix(),
				})
			}
		case *events.ConnectFailure:
			log.Conn("failed", deviceID, currentJID)
		case *events.PushNameSetting:
			sendAvailablePresence(currentJID, deviceID)
		case *events.AppStateSyncComplete:
			if appStateWebhookEnabled {
				dispatchWebhook(deviceID, webhook.EventAppStateSyncComplete, map[string]interface{}{
					"jid":  currentJID,
					"name": e.Name,
				})
			}
		case *events.AppState:
			if appStateWebhookEnabled {
				dispatchWebhook(deviceID, webhook.EventAppStatePatchReceived, map[string]interface{}{
					"jid":   currentJID,
					"index": e.Index,
				})
			}
		// Call events
		case *events.CallOffer:
			log.Call("offer", deviceID, e.CallID, e.From.String())
			dispatchWebhook(deviceID, webhook.EventCallOffer, map[string]interface{}{
				"jid":       currentJID,
				"call_id":   e.CallID,
				"from":      e.From.String(),
				"timestamp": e.Timestamp.Unix(),
			})
		case *events.CallAccept:
			log.Call("accept", deviceID, e.CallID, e.From.String())
			dispatchWebhook(deviceID, webhook.EventCallAccept, map[string]interface{}{
				"jid":       currentJID,
				"call_id":   e.CallID,
				"from":      e.From.String(),
				"timestamp": e.Timestamp.Unix(),
			})
		case *events.CallTerminate:
			log.Call("end", deviceID, e.CallID, e.From.String())
			dispatchWebhook(deviceID, webhook.EventCallTerminate, map[string]interface{}{
				"jid":       currentJID,
				"call_id":   e.CallID,
				"from":      e.From.String(),
				"timestamp": e.Timestamp.Unix(),
				"reason":    e.Reason,
			})
		// History sync events
		case *events.HistorySync:
			dispatchWebhook(deviceID, webhook.EventHistorySync, map[string]interface{}{
				"jid":                currentJID,
				"sync_type":          e.Data.GetSyncType().String(),
				"progress":           e.Data.GetProgress(),
				"conversation_count": len(e.Data.GetConversations()),
			})
		// Blocklist change events
		case *events.Blocklist:
			dispatchWebhook(deviceID, webhook.EventBlocklistChange, map[string]interface{}{
				"jid":    currentJID,
				"action": string(e.Action),
			})
		// Group join events
		case *events.JoinedGroup:
			log.Grp("joined", deviceID, e.JID.String())
			dispatchWebhook(deviceID, webhook.EventGroupJoin, map[string]interface{}{
				"jid":       currentJID,
				"group_jid": e.JID.String(),
				"reason":    e.Reason,
			})
		// Group participant update events
		case *events.GroupInfo:
			log.Grp("updated", deviceID, e.JID.String())
			dispatchWebhook(deviceID, webhook.EventGroupInfoUpdate, map[string]interface{}{
				"jid":       currentJID,
				"group_jid": e.JID.String(),
				"sender":    e.Sender.String(),
				"timestamp": e.Timestamp.Unix(),
			})
		// Contact update events
		case *events.Contact:
			dispatchWebhook(deviceID, webhook.EventContactUpdate, map[string]interface{}{
				"jid":          currentJID,
				"contact_jid":  e.JID.String(),
				"action":       e.Action.String(),
			})
		// Newsletter events
		case *events.NewsletterJoin:
			dispatchWebhook(deviceID, webhook.EventNewsletterJoin, map[string]interface{}{
				"jid":            currentJID,
				"newsletter_jid": e.ID.String(),
			})
		case *events.NewsletterLeave:
			dispatchWebhook(deviceID, webhook.EventNewsletterLeave, map[string]interface{}{
				"jid":            currentJID,
				"newsletter_jid": e.ID.String(),
				"role":           e.Role,
			})
		case *events.NewsletterMuteChange:
			dispatchWebhook(deviceID, webhook.EventNewsletterUpdate, map[string]interface{}{
				"jid":            currentJID,
				"newsletter_jid": e.ID.String(),
				"mute":           e.Mute,
			})
		}
	}
}

func dispatchWebhook(deviceID string, eventType webhook.EventType, data map[string]interface{}) {
	if webhookEngine == nil {
		return
	}
	webhookEngine.Dispatch(context.Background(), deviceID, webhook.WebhookEvent{
		EventType: eventType,
		DeviceID:  deviceID,
		Timestamp: time.Now(),
		Data:      data,
	})
}

func sendAvailablePresence(jid string, deviceID string) {
	client, err := currentClient(jid, deviceID)
	if err != nil || client == nil {
		return
	}
	if client.Store != nil && len(client.Store.PushName) == 0 {
		return
	}
	_ = client.SendPresence(context.Background(), types.PresenceAvailable)
}

func autoMarkMessageAsRead(jid string, deviceID string, evt *events.Message) {
	if !autoMarkReadEnabled || evt == nil {
		return
	}
	if evt.Info.IsFromMe || evt.Info.Chat == types.StatusBroadcastJID || strings.HasSuffix(evt.Info.Chat.String(), "@broadcast") {
		return
	}
	client, err := currentClient(jid, deviceID)
	if err != nil || client == nil {
		return
	}
	if readReceiptJitterEnabled {
		delay := jitterDuration(readReceiptDelayMin, readReceiptDelayMax)
		time.Sleep(delay)
	}
	err = client.MarkRead(context.Background(), []types.MessageID{evt.Info.ID}, time.Now(), evt.Info.Chat, evt.Info.Sender)
	if err != nil {
		log.EvtErr("msg", "mark-read", deviceID, err)
	}
}

func purgeDeviceSession(ctx context.Context, deviceID string) {
	if ctx == nil {
		ctx = context.Background()
	}
	if WhatsAppDatastore == nil {
		return
	}
	storedJID, _, err := GetWhatsMeowJID(ctx, deviceID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.EvtErr("cleanup", "get-jid", deviceID, err)
	}
	devices, err := WhatsAppDatastore.GetAllDevices(ctx)
	if err != nil {
		log.EvtErr("cleanup", "list-devices", deviceID, err)
		return
	}
	for _, dev := range devices {
		if dev.ID == nil {
			continue
		}
		if storedJID != "" && dev.ID.String() != storedJID {
			continue
		}
		if err := WhatsAppDatastore.DeleteDevice(ctx, dev); err != nil {
			log.EvtErr("cleanup", "del-device", deviceID, err)
		}
	}
}

func GetWebhookEngine() *webhook.Engine {
	return webhookEngine
}

func WhatsAppGenerateQR(ctx context.Context, qrChan <-chan whatsmeow.QRChannelItem) (string, int, bool, error) {
	for {
		select {
		case <-ctx.Done():
			return "", 0, false, ctx.Err()
		case evt, ok := <-qrChan:
			if !ok {
				return "", 0, false, errors.New("whatsapp qr channel closed before delivering a code")
			}
			switch {
			case evt.Event == "code":
				qrPNG, err := qrCode.Encode(evt.Code, qrCode.Medium, 256)
				if err != nil {
					return "", 0, false, err
				}
				timeout := int(evt.Timeout.Seconds())
				return base64.StdEncoding.EncodeToString(qrPNG), timeout, false, nil
			case evt.Event == whatsmeow.QRChannelSuccess.Event:
				return "", 0, true, nil
			case evt.Event == whatsmeow.QRChannelTimeout.Event:
				return "", 0, false, errors.New("whatsapp qr channel timed out")
			case evt.Event == whatsmeow.QRChannelErrUnexpectedEvent.Event:
				return "", 0, false, errors.New("whatsapp qr channel entered an unexpected state")
			case evt.Event == whatsmeow.QRChannelClientOutdated.Event:
				return "", 0, false, errors.New("whatsapp client version is outdated for QR pairing")
			case evt.Event == whatsmeow.QRChannelScannedWithoutMultidevice.Event:
				return "", 0, false, errors.New("whatsapp qr scanned without multi-device enabled")
			case evt.Event == "error":
				if evt.Error != nil {
					return "", 0, false, evt.Error
				}
				return "", 0, false, errors.New("whatsapp qr channel reported an unspecified error")
			}
		}
	}
}

func consumeQRChannel(ctx context.Context, qrChan <-chan whatsmeow.QRChannelItem, cancel context.CancelFunc, jid string, deviceID string) {
	go func() {
		defer cancel()
		masked := maskJIDForLog(jid)
		if masked == "" {
			masked = "unknown"
		}
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-qrChan:
				if !ok {
					return
				}
				switch evt.Event {
				case whatsmeow.QRChannelSuccess.Event:
					client := getClient(jid, deviceID)
					if client != nil && client.Store.ID != nil {
						newJID := WhatsAppDecomposeJID(client.Store.ID.User)
						_ = SaveDeviceRouting(context.Background(), deviceID, client.Store.ID.String())
						log.EvtOK("qr", "paired", deviceID, newJID)
					} else {
						log.EvtOK("qr", "paired", deviceID)
					}
					return
				case whatsmeow.QRChannelTimeout.Event:
					log.Evt("qr", "timeout", deviceID)
					return
				case whatsmeow.QRChannelErrUnexpectedEvent.Event:
					log.Evt("qr", "unexpected", deviceID)
					return
				case whatsmeow.QRChannelClientOutdated.Event:
					log.Evt("qr", "outdated", deviceID)
					return
				case whatsmeow.QRChannelScannedWithoutMultidevice.Event:
					log.Evt("qr", "no-multidevice", deviceID)
					return
				case "error":
					log.Evt("qr", "error", deviceID)
					return
				case "code":
					// QR code refresh - silent
				default:
					// Other events - silent
				}
			}
		}
	}()
}

func WhatsAppLogin(jid string, deviceID string) (string, int, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", 0, err
	}

	client.Disconnect()

	if client.Store.ID == nil {
		ctx, cancel := context.WithTimeout(context.Background(), qrChannelWaitTimeout)

		qrChanGenerate, err := client.GetQRChannel(ctx)
		if err != nil {
			cancel()
			return "", 0, err
		}

		err = client.Connect()
		if err != nil {
			cancel()
			return "", 0, err
		}

		qrImage, qrTimeout, paired, err := WhatsAppGenerateQR(ctx, qrChanGenerate)
		if err != nil {
			cancel()
			return "", 0, err
		}
		if paired {
			cancel()
			return "WhatsApp Client is already paired", 0, nil
		}

		consumeQRChannel(ctx, qrChanGenerate, cancel, jid, deviceID)

		return "data:image/png;base64," + qrImage, qrTimeout, nil
	}

	err = WhatsAppReconnect(jid, deviceID)
	if err != nil {
		return "", 0, err
	}

	return "WhatsApp Client is Reconnected", 0, nil
}

func WhatsAppLoginPair(jid string, deviceID string, phone string) (string, int, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", 0, err
	}

	client.Disconnect()

	if client.Store.ID == nil {
		ctx, cancel := context.WithTimeout(context.Background(), pairPhoneRequestTimeout)
		defer cancel()

		err = client.Connect()
		if err != nil {
			return "", 0, err
		}

		code, err := client.PairPhone(ctx, phone, true, whatsmeow.PairClientChrome, "Chrome ("+runtime.GOOS+")")
		if err != nil {
			return "", 0, err
		}

		return code, 160, nil
	}

	err = WhatsAppReconnect(jid, deviceID)
	if err != nil {
		return "", 0, err
	}

	return "WhatsApp Client is Reconnected", 0, nil
}

func WhatsAppReconnect(jid string, deviceID string) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}

	client.Disconnect()

	if client.Store.ID != nil {
		err = client.Connect()
		if err != nil {
			return err
		}

		_ = SaveDeviceRouting(context.Background(), deviceID, client.Store.ID.String())
		newJID := WhatsAppDecomposeJID(client.Store.ID.User)
		log.Conn("reconnected", deviceID, newJID)
		return nil
	}

	return errors.New("WhatsApp Client Store ID is Empty, Please Re-Login and Scan QR Code Again")
}

func WhatsAppLogout(jid string, deviceID string) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}

	if client.Store.ID != nil {
		WhatsAppPresence(context.Background(), jid, deviceID, false)

		logoutCtx, logoutCancel := context.WithTimeout(context.Background(), logoutRequestTimeout)
		defer logoutCancel()

		err = client.Logout(logoutCtx)
		if err != nil {
			client.Disconnect()
			storeCtx, storeCancel := context.WithTimeout(context.Background(), routingCleanupTimeout)
			defer storeCancel()
			err = client.Store.Delete(storeCtx)
			if err != nil {
				return err
			}
		}

		routingCtx, routingCancel := context.WithTimeout(context.Background(), routingCleanupTimeout)
		defer routingCancel()
		err = DeleteDeviceRouting(routingCtx, deviceID)
		if err != nil {
			return err
		}

		deleteClient(jid, deviceID)

		return nil
	}

	return errors.New("WhatsApp Client Store ID is Empty, Please Re-Login and Scan QR Code Again")
}

func WhatsAppIsClientOK(jid string, deviceID string) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}

	if !client.IsConnected() {
		return errors.New("WhatsApp Client is not Connected")
	}

	if !client.IsLoggedIn() {
		return errors.New("WhatsApp Client is not Logged In")
	}

	return nil
}

func WhatsAppGetJID(ctx context.Context, jid string, deviceID string, id string) types.JID {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return types.EmptyJID
	}
	normalized := WhatsAppDecomposeJID(id)
	if normalized == "" {
		return types.EmptyJID
	}
	// Only enforce registration for personal JIDs; groups/newsletters are skipped
	parsed := WhatsAppComposeJID(id)
	if parsed.Server == types.GroupServer {
		return parsed
	}

	if err := validatePersonalPhoneInput(normalized); err != nil {
		return types.EmptyJID
	}

	if cached, ok := getIsOnCache(normalized); ok {
		return cached
	}

	jidVal := lookupIsOnWhatsApp(ctx, client, normalized)
	setIsOnCache(normalized, jidVal)
	return jidVal
}

func WhatsAppCheckJID(ctx context.Context, jid string, deviceID string, id string) (types.JID, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	_, err := currentClient(jid, deviceID)
	if err != nil {
		return types.EmptyJID, err
	}
	if err := validatePersonalPhoneInput(id); err != nil {
		return types.EmptyJID, err
	}
	remoteJID := WhatsAppComposeJID(id)
	if remoteJID.Server != types.GroupServer {
		resolved := WhatsAppGetJID(ctx, jid, deviceID, id)
		if resolved.IsEmpty() {
			return types.EmptyJID, errors.New("WhatsApp Personal ID is Not Registered")
		}
		remoteJID = resolved
	}
	return remoteJID, nil
}

func WhatsAppComposeJID(id string) types.JID {
	// First try to parse the full JID directly (e.g., "xxx@g.us", "xxx@s.whatsapp.net")
	// This preserves the server type from the input
	if strings.ContainsRune(id, '@') {
		if parsed, err := types.ParseJID(id); err == nil && parsed.User != "" {
			return parsed
		}
	}

	// For inputs without @ or failed parsing, extract the user part
	id = WhatsAppDecomposeJID(id)

	// Group JIDs have a hyphen (regular groups) or are 18+ digits (channels/newsletters)
	if strings.ContainsRune(id, '-') || len(id) >= 18 {
		return types.NewJID(id, types.GroupServer)
	}
	return types.NewJID(id, types.DefaultUserServer)
}

func WhatsAppDecomposeJID(id string) string {
	if strings.ContainsRune(id, '@') {
		buffers := strings.Split(id, "@")
		id = buffers[0]
	}

	if len(id) > 0 && id[0] == '+' {
		id = id[1:]
	}

	return strings.TrimSpace(id)
}

func validatePersonalPhoneInput(id string) error {
	if strings.ContainsRune(id, '@') {
		return nil
	}
	trimmed := strings.TrimSpace(id)
	if strings.Contains(trimmed, "-") {
		// Likely a group/newsletter ID, skip strict phone validation
		return nil
	}
	if trimmed == "" {
		return errors.New("Phone number cannot be empty")
	}
	if trimmed[0] == '+' {
		trimmed = trimmed[1:]
	}
	if strings.HasPrefix(trimmed, "0") {
		return errors.New("Phone number must be in international format without leading 0")
	}
	if !phonePattern.MatchString(trimmed) {
		return errors.New("Phone number must be digits only and at least 6 characters")
	}
	return nil
}

func getIsOnCache(normalized string) (types.JID, bool) {
	if !isOnCacheEnabled {
		return types.EmptyJID, false
	}
	isOnCacheMu.RLock()
	entry, ok := isOnCache[normalized]
	isOnCacheMu.RUnlock()
	if !ok {
		return types.EmptyJID, false
	}
	if time.Now().After(entry.expires) {
		isOnCacheMu.Lock()
		delete(isOnCache, normalized)
		isOnCacheMu.Unlock()
		return types.EmptyJID, false
	}
	if !entry.ok {
		return types.EmptyJID, true
	}
	return entry.jid, true
}

func setIsOnCache(normalized string, jid types.JID) {
	if !isOnCacheEnabled {
		return
	}
	entry := isOnCacheEntry{
		jid:     jid,
		ok:      !jid.IsEmpty(),
		expires: time.Now().Add(isOnCacheTTL),
	}
	isOnCacheMu.Lock()
	if len(isOnCache) >= isOnCacheMax {
		// simple eviction: remove one arbitrary entry
		for k := range isOnCache {
			delete(isOnCache, k)
			break
		}
	}
	isOnCache[normalized] = entry
	isOnCacheMu.Unlock()
}

func lookupIsOnWhatsApp(ctx context.Context, client *whatsmeow.Client, normalized string) types.JID {
	key := "+" + normalized

	res, err, _ := isOnSingleFlight.Do(key, func() (interface{}, error) {
		infos, err := client.IsOnWhatsApp(ctx, []string{key})
		if err != nil {
			return nil, err
		}
		if len(infos) == 0 || !infos[0].IsIn {
			return types.EmptyJID, nil
		}
		return infos[0].JID, nil
	})
	if err != nil {
		return types.EmptyJID
	}
	if jidVal, ok := res.(types.JID); ok {
		return jidVal
	}
	return types.EmptyJID
}

func detectMime(payload []byte, hinted string) string {
	if hinted != "" && hinted != "application/octet-stream" {
		return hinted
	}
	return http.DetectContentType(payload)
}

func enforceSizeLimit(name string, size int64, limit int64) error {
	if limit == 0 {
		return nil
	}
	if size > limit {
		return fmt.Errorf("%s exceeds maximum allowed size (%s > %s)", name, formatBytes(size), formatBytes(limit))
	}
	return nil
}

func formatBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%dB", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func WhatsAppPresence(ctx context.Context, jid string, deviceID string, isAvailable bool) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return
	}
	if isAvailable {
		_ = client.SendPresence(ctx, types.PresenceAvailable)
	} else {
		_ = client.SendPresence(ctx, types.PresenceUnavailable)
	}
}

func WhatsAppComposeStatus(ctx context.Context, jid string, deviceID string, rjid types.JID, isComposing bool, isAudio bool) {
	if ctx == nil {
		ctx = context.Background()
	}
	var typeCompose types.ChatPresence
	if isComposing {
		typeCompose = types.ChatPresenceComposing
	} else {
		typeCompose = types.ChatPresencePaused
	}

	var typeComposeMedia types.ChatPresenceMedia
	if isAudio {
		typeComposeMedia = types.ChatPresenceMediaAudio
	} else {
		typeComposeMedia = types.ChatPresenceMediaText
	}

	client, err := currentClient(jid, deviceID)
	if err != nil {
		return
	}
	_ = client.SendChatPresence(ctx, rjid, typeCompose, typeComposeMedia)
}

func WhatsAppMessageDelete(ctx context.Context, jid string, deviceID string, rjid string, msgid string) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}
	remoteJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, rjid)
	if err != nil {
		return err
	}
	WhatsAppPresence(context.Background(), jid, deviceID, true)
	WhatsAppComposeStatus(context.Background(), jid, deviceID, remoteJID, true, false)
	defer func() {
		WhatsAppComposeStatus(context.Background(), jid, deviceID, remoteJID, false, false)
		WhatsAppPresence(context.Background(), jid, deviceID, false)
	}()
	_, err = client.SendMessage(ctx, remoteJID, client.BuildRevoke(remoteJID, types.EmptyJID, msgid))
	return err
}

func WhatsAppGroupGet(jid string, deviceID string) ([]types.GroupInfo, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}
	groups, err := client.GetJoinedGroups(context.Background())
	if err != nil {
		return nil, err
	}
	var gids []types.GroupInfo
	for _, group := range groups {
		gids = append(gids, *group)
	}
	return gids, nil
}

func WhatsAppGroupGetWithMembers(ctx context.Context, jid string, deviceID string) ([]EnhancedGroupInfo, error) {
	return WhatsAppGroupList(ctx, jid, deviceID, true, false)
}

// WhatsAppGroupList provides a tunable group listing helper that supports optional
// phone-number resolution and caching.
// OPTIMIZED: Uses parallel processing and skips slow LID resolution by default
func WhatsAppGroupList(ctx context.Context, jid string, deviceID string, resolvePhoneNumbers bool, forceRefresh bool) ([]EnhancedGroupInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Check cache first (fast path)
	if !forceRefresh {
		if cached, ok := loadGroupListCache(deviceID, resolvePhoneNumbers); ok {
			return cached, nil
		}
	}

	// OPTIMIZATION: Singleflight pattern - deduplicate concurrent requests for same device
	inflightKey := fmt.Sprintf("%s:%v", deviceID, resolvePhoneNumbers)

	groupListInflightMu.Lock()
	if inflight, exists := groupListInflight[inflightKey]; exists {
		groupListInflightMu.Unlock()
		<-inflight.done
		if inflight.err != nil {
			return nil, inflight.err
		}
		return inflight.result, nil
	}

	inflight := &inflightRequest{done: make(chan struct{})}
	groupListInflight[inflightKey] = inflight
	groupListInflightMu.Unlock()

	defer func() {
		groupListInflightMu.Lock()
		delete(groupListInflight, inflightKey)
		groupListInflightMu.Unlock()
		close(inflight.done)
	}()

	client, err := currentClient(jid, deviceID)
	if err != nil {
		inflight.err = err
		return nil, err
	}

	if jid == "" {
		if client.Store != nil && client.Store.ID != nil {
			jid = WhatsAppDecomposeJID(client.Store.ID.User)
		} else {
			inflight.err = errors.New("JID is empty and client store has no JID")
			return nil, inflight.err
		}
	}

	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		inflight.err = err
		return nil, err
	}

	groupCtx, groupCancel := context.WithTimeout(ctx, groupFetchTimeout)
	defer groupCancel()

	joinedGroups, err := client.GetJoinedGroups(groupCtx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			inflight.err = fmt.Errorf("GetJoinedGroups timed out: %w", err)
			return nil, inflight.err
		}
		inflight.err = fmt.Errorf("GetJoinedGroups failed: %w", err)
		return nil, inflight.err
	}

	var result []EnhancedGroupInfo

	if len(joinedGroups) > 5 {
		result, err = ConvertGroupsInParallelWithContext(ctx, joinedGroups, nil, groupConversionWorkers)
		if err != nil {
			inflight.err = err
			return nil, err
		}
	} else {
		result = make([]EnhancedGroupInfo, 0, len(joinedGroups))
		for _, group := range joinedGroups {
			if group == nil {
				continue
			}
			enhanced := ConvertToEnhancedGroupInfo(*group, nil)
			result = append(result, enhanced)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].GroupCreated.After(result[j].GroupCreated)
	})

	storeGroupListCache(deviceID, resolvePhoneNumbers, result)
	inflight.result = result

	return result, nil
}

func WhatsAppGroupCreate(ctx context.Context, jid string, deviceID string, subject string, participantIDs []string) (*types.GroupInfo, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	req := whatsmeow.ReqCreateGroup{
		Name: subject,
	}
	if len(participantIDs) > 0 {
		participants := make([]types.JID, 0, len(participantIDs))
		for _, participant := range participantIDs {
			parsed, err := WhatsAppCheckJID(context.Background(), jid, deviceID, participant)
			if err != nil {
				return nil, err
			}
			if parsed.Server == types.GroupServer {
				return nil, ErrParticipantMustBeUser
			}
			participants = append(participants, parsed)
		}
		req.Participants = participants
	}
	group, err := client.CreateGroup(ctx, req)
	if err != nil {
		return nil, err
	}
	invalidateGroupListCache(deviceID)
	return group, nil
}

func WhatsAppGroupJoin(ctx context.Context, jid string, deviceID string, link string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}
	gid, err := client.JoinGroupWithLink(ctx, link)
	if err != nil {
		return "", err
	}
	invalidateGroupListCache(deviceID)
	return gid.String(), nil
}

func WhatsAppGroupLeave(ctx context.Context, jid string, deviceID string, gjid string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}
	groupJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, gjid)
	if err != nil {
		return err
	}
	if groupJID.Server != types.GroupServer {
		return ErrInvalidGroupID
	}
	err = client.LeaveGroup(ctx, groupJID)
	if err == nil {
		invalidateGroupListCache(deviceID)
	}
	return err
}

func WhatsAppGroupSetName(ctx context.Context, jid string, deviceID string, gjid string, name string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}
	groupJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, gjid)
	if err != nil {
		return err
	}
	if groupJID.Server != types.GroupServer {
		return ErrInvalidGroupID
	}
	err = client.SetGroupName(ctx, groupJID, name)
	if err == nil {
		invalidateGroupListCache(deviceID)
	}
	return err
}

func WhatsAppGroupSetDescription(ctx context.Context, jid string, deviceID string, gjid string, description string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}
	groupJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, gjid)
	if err != nil {
		return err
	}
	if groupJID.Server != types.GroupServer {
		return ErrInvalidGroupID
	}
	err = client.SetGroupDescription(ctx, groupJID, description)
	if err == nil {
		invalidateGroupListCache(deviceID)
	}
	return err
}

func WhatsAppGroupSetPhoto(ctx context.Context, jid string, deviceID string, gjid string, photo []byte) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}
	groupJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, gjid)
	if err != nil {
		return "", err
	}
	if groupJID.Server != types.GroupServer {
		return "", ErrInvalidGroupID
	}
	photoID, err := client.SetGroupPhoto(ctx, groupJID, photo)
	if err != nil {
		return "", err
	}
	invalidateGroupListCache(deviceID)
	return photoID, nil
}

func WhatsAppGroupInviteLink(ctx context.Context, jid string, deviceID string, gjid string, reset bool) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}
	groupJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, gjid)
	if err != nil {
		return "", err
	}
	if groupJID.Server != types.GroupServer {
		return "", ErrInvalidGroupID
	}
	return client.GetGroupInviteLink(ctx, groupJID, reset)
}

func WhatsAppGroupGetRequestParticipants(ctx context.Context, jid string, deviceID string, gjid string) ([]types.GroupParticipantRequest, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}
	groupJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, gjid)
	if err != nil {
		return nil, err
	}
	if groupJID.Server != types.GroupServer {
		return nil, ErrInvalidGroupID
	}
	return client.GetGroupRequestParticipants(ctx, groupJID)
}

func WhatsAppGroupSetLocked(jid string, deviceID string, gjid string, locked bool) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}
	groupJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, gjid)
	if err != nil {
		return err
	}
	if groupJID.Server != types.GroupServer {
		return ErrInvalidGroupID
	}
	err = client.SetGroupLocked(context.Background(), groupJID, locked)
	if err == nil {
		invalidateGroupListCache(deviceID)
	}
	return err
}

func WhatsAppGroupSetAnnounce(jid string, deviceID string, gjid string, announce bool) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}
	groupJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, gjid)
	if err != nil {
		return err
	}
	if groupJID.Server != types.GroupServer {
		return ErrInvalidGroupID
	}
	err = client.SetGroupAnnounce(context.Background(), groupJID, announce)
	if err == nil {
		invalidateGroupListCache(deviceID)
	}
	return err
}

func WhatsAppGroupSetJoinApprovalMode(jid string, deviceID string, gjid string, mode bool) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}
	groupJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, gjid)
	if err != nil {
		return err
	}
	if groupJID.Server != types.GroupServer {
		return ErrInvalidGroupID
	}
	err = client.SetGroupJoinApprovalMode(context.Background(), groupJID, mode)
	if err == nil {
		invalidateGroupListCache(deviceID)
	}
	return err
}

func WhatsAppSendText(ctx context.Context, jid string, deviceID string, rjid string, message string, opts *SendOptions) (string, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}
	if strings.TrimSpace(message) == "" {
		return "", errors.New("Message cannot be empty")
	}
	remoteJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, rjid)
	if err != nil {
		return "", err
	}
	msgExtra := whatsmeow.SendRequestExtra{ID: client.GenerateMessageID()}
	msgContent := &waE2E.Message{
		Conversation: proto.String(message),
	}
	if err := waitRateLimit(ctx, deviceID); err != nil {
		return "", err
	}
	cleanup := beginPresenceSimulation(ctx, jid, deviceID, remoteJID, false, opts)
	defer cleanup()
	_, err = client.SendMessage(ctx, remoteJID, msgContent, msgExtra)
	if err != nil {
		return "", err
	}
	return msgExtra.ID, nil
}

func WhatsAppSendDocument(ctx context.Context, jid string, deviceID string, rjid string, documentBytes []byte, documentType string, documentName string, opts *SendOptions) (string, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}
	if len(documentBytes) == 0 {
		return "", errors.New("Document payload cannot be empty")
	}
	if err := enforceSizeLimit("document", int64(len(documentBytes)), maxDocumentBytes); err != nil {
		return "", err
	}
	remoteJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, rjid)
	if err != nil {
		return "", err
	}
	if err := waitRateLimit(ctx, deviceID); err != nil {
		return "", err
	}
	cleanup := beginPresenceSimulation(ctx, jid, deviceID, remoteJID, false, opts)
	defer cleanup()
	documentMime := detectMime(documentBytes, documentType)
	allowedDocumentMimes := map[string]bool{
		"application/pdf":    true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/vnd.ms-excel": true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
		"application/vnd.ms-powerpoint":                                             true,
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
		"text/plain":               true,
		"application/zip":          true,
		"application/octet-stream": true,
	}
	if !allowedDocumentMimes[documentMime] {
		return "", fmt.Errorf("Document MIME type %s is not allowed", documentMime)
	}
	documentUploaded, err := client.Upload(ctx, documentBytes, whatsmeow.MediaDocument)
	if err != nil {
		return "", errors.New("Error While Uploading Media to WhatsApp Server")
	}
	msgExtra := whatsmeow.SendRequestExtra{ID: client.GenerateMessageID()}
	msgContent := &waE2E.Message{
		DocumentMessage: &waE2E.DocumentMessage{
			URL:           proto.String(documentUploaded.URL),
			DirectPath:    proto.String(documentUploaded.DirectPath),
			Mimetype:      proto.String(documentMime),
			FileName:      proto.String(documentName),
			FileLength:    proto.Uint64(documentUploaded.FileLength),
			FileSHA256:    documentUploaded.FileSHA256,
			FileEncSHA256: documentUploaded.FileEncSHA256,
			MediaKey:      documentUploaded.MediaKey,
		},
	}
	_, err = client.SendMessage(ctx, remoteJID, msgContent, msgExtra)
	if err != nil {
		return "", err
	}
	return msgExtra.ID, nil
}

func WhatsAppSendImage(ctx context.Context, jid string, deviceID string, rjid string, imageBytes []byte, imageType string, imageCaption string, isViewOnce bool, opts *SendOptions) (string, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}
	if len(imageBytes) == 0 {
		return "", errors.New("Image payload cannot be empty")
	}
	if err := enforceSizeLimit("image", int64(len(imageBytes)), maxImageBytes); err != nil {
		return "", err
	}
	imageMime := detectMime(imageBytes, imageType)
	allowedImageMimes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/webp": true,
	}
	if !allowedImageMimes[imageMime] {
		return "", fmt.Errorf("Image MIME type %s is not allowed", imageMime)
	}
	remoteJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, rjid)
	if err != nil {
		return "", err
	}
	if err := waitRateLimit(ctx, deviceID); err != nil {
		return "", err
	}
	cleanup := beginPresenceSimulation(ctx, jid, deviceID, remoteJID, false, opts)
	defer cleanup()
	isWhatsAppImageConvertWebP, err := env.GetEnvBool("WHATSAPP_MEDIA_IMAGE_CONVERT_WEBP")
	if err != nil {
		isWhatsAppImageConvertWebP = false
	}
	if imageType == "image/webp" && isWhatsAppImageConvertWebP {
		imgConvDecode, err := imgconv.Decode(bytes.NewReader(imageBytes))
		if err != nil {
			return "", errors.New("Error While Decoding Convert Image Stream")
		}
		imgConvEncode := new(bytes.Buffer)
		err = imgconv.Write(imgConvEncode, imgConvDecode, &imgconv.FormatOption{Format: imgconv.PNG})
		if err != nil {
			return "", errors.New("Error While Encoding Convert Image Stream")
		}
		imageBytes = imgConvEncode.Bytes()
		imageType = "image/png"
	}
	isWhatsAppImageCompression, err := env.GetEnvBool("WHATSAPP_MEDIA_IMAGE_COMPRESSION")
	if err != nil {
		isWhatsAppImageCompression = false
	}
	if isWhatsAppImageCompression {
		imgResizeDecode, err := imgconv.Decode(bytes.NewReader(imageBytes))
		if err != nil {
			return "", errors.New("Error While Decoding Resize Image Stream")
		}
		imgResizeEncode := new(bytes.Buffer)
		err = imgconv.Write(imgResizeEncode,
			imgconv.Resize(imgResizeDecode, &imgconv.ResizeOption{Width: 1024}),
			&imgconv.FormatOption{})
		if err != nil {
			return "", errors.New("Error While Encoding Resize Image Stream")
		}
		imageBytes = imgResizeEncode.Bytes()
	}
	imageMime = detectMime(imageBytes, imageMime)
	if !allowedImageMimes[imageMime] {
		return "", fmt.Errorf("Image MIME type %s is not allowed", imageMime)
	}
	imgThumbDecode, err := imgconv.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return "", errors.New("Error While Decoding Thumbnail Image Stream")
	}
	imgThumbEncode := new(bytes.Buffer)
	err = imgconv.Write(imgThumbEncode,
		imgconv.Resize(imgThumbDecode, &imgconv.ResizeOption{Width: 72}),
		&imgconv.FormatOption{Format: imgconv.JPEG})
	if err != nil {
		return "", errors.New("Error While Encoding Thumbnail Image Stream")
	}
	imageUploaded, err := client.Upload(ctx, imageBytes, whatsmeow.MediaImage)
	if err != nil {
		return "", errors.New("Error While Uploading Media to WhatsApp Server")
	}
	imageThumbUploaded, err := client.Upload(ctx, imgThumbEncode.Bytes(), whatsmeow.MediaLinkThumbnail)
	if err != nil {
		return "", errors.New("Error while Uploading Image Thumbnail to WhatsApp Server")
	}
	msgExtra := whatsmeow.SendRequestExtra{
		ID: client.GenerateMessageID(),
	}
	msgContent := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			URL:                 proto.String(imageUploaded.URL),
			DirectPath:          proto.String(imageUploaded.DirectPath),
			Mimetype:            proto.String(imageMime),
			Caption:             proto.String(imageCaption),
			FileLength:          proto.Uint64(imageUploaded.FileLength),
			FileSHA256:          imageUploaded.FileSHA256,
			FileEncSHA256:       imageUploaded.FileEncSHA256,
			MediaKey:            imageUploaded.MediaKey,
			JPEGThumbnail:       imgThumbEncode.Bytes(),
			ThumbnailDirectPath: &imageThumbUploaded.DirectPath,
			ThumbnailSHA256:     imageThumbUploaded.FileSHA256,
			ThumbnailEncSHA256:  imageThumbUploaded.FileEncSHA256,
			ViewOnce:            proto.Bool(isViewOnce),
		},
	}
	_, err = client.SendMessage(ctx, remoteJID, msgContent, msgExtra)
	if err != nil {
		return "", err
	}
	return msgExtra.ID, nil
}

// Status/Stories functions

func WhatsAppPostTextStatus(ctx context.Context, jid string, deviceID string, text string, backgroundColor string, font int) (string, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}
	if strings.TrimSpace(text) == "" {
		return "", errors.New("Status text cannot be empty")
	}
	if err := waitRateLimit(ctx, deviceID); err != nil {
		return "", err
	}
	// Status JID for posting to status/stories
	statusJID := types.StatusBroadcastJID
	// Build the extended text message for status
	msgContent := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(text),
		},
	}
	msgExtra := whatsmeow.SendRequestExtra{ID: client.GenerateMessageID()}
	_, err = client.SendMessage(ctx, statusJID, msgContent, msgExtra)
	if err != nil {
		return "", err
	}
	return msgExtra.ID, nil
}

func WhatsAppPostImageStatus(ctx context.Context, jid string, deviceID string, imageBytes []byte, caption string) (string, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}
	if len(imageBytes) == 0 {
		return "", errors.New("Image payload cannot be empty")
	}
	if err := enforceSizeLimit("image", int64(len(imageBytes)), maxImageBytes); err != nil {
		return "", err
	}
	if err := waitRateLimit(ctx, deviceID); err != nil {
		return "", err
	}
	statusJID := types.StatusBroadcastJID
	imageMime := detectMime(imageBytes, "image/jpeg")
	imageUploaded, err := client.Upload(ctx, imageBytes, whatsmeow.MediaImage)
	if err != nil {
		return "", errors.New("Error while uploading image to WhatsApp server")
	}
	msgExtra := whatsmeow.SendRequestExtra{ID: client.GenerateMessageID()}
	msgContent := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			URL:           proto.String(imageUploaded.URL),
			DirectPath:    proto.String(imageUploaded.DirectPath),
			Mimetype:      proto.String(imageMime),
			Caption:       proto.String(caption),
			FileLength:    proto.Uint64(imageUploaded.FileLength),
			FileSHA256:    imageUploaded.FileSHA256,
			FileEncSHA256: imageUploaded.FileEncSHA256,
			MediaKey:      imageUploaded.MediaKey,
		},
	}
	_, err = client.SendMessage(ctx, statusJID, msgContent, msgExtra)
	if err != nil {
		return "", err
	}
	return msgExtra.ID, nil
}

func WhatsAppPostVideoStatus(ctx context.Context, jid string, deviceID string, videoBytes []byte, caption string) (string, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}
	if len(videoBytes) == 0 {
		return "", errors.New("Video payload cannot be empty")
	}
	if err := enforceSizeLimit("video", int64(len(videoBytes)), maxVideoBytes); err != nil {
		return "", err
	}
	if err := waitRateLimit(ctx, deviceID); err != nil {
		return "", err
	}
	statusJID := types.StatusBroadcastJID
	videoMime := detectMime(videoBytes, "video/mp4")
	videoUploaded, err := client.Upload(ctx, videoBytes, whatsmeow.MediaVideo)
	if err != nil {
		return "", errors.New("Error while uploading video to WhatsApp server")
	}
	msgExtra := whatsmeow.SendRequestExtra{ID: client.GenerateMessageID()}
	msgContent := &waE2E.Message{
		VideoMessage: &waE2E.VideoMessage{
			URL:           proto.String(videoUploaded.URL),
			DirectPath:    proto.String(videoUploaded.DirectPath),
			Mimetype:      proto.String(videoMime),
			Caption:       proto.String(caption),
			FileLength:    proto.Uint64(videoUploaded.FileLength),
			FileSHA256:    videoUploaded.FileSHA256,
			FileEncSHA256: videoUploaded.FileEncSHA256,
			MediaKey:      videoUploaded.MediaKey,
		},
	}
	_, err = client.SendMessage(ctx, statusJID, msgContent, msgExtra)
	if err != nil {
		return "", err
	}
	return msgExtra.ID, nil
}

func WhatsAppGetStatusUpdates(ctx context.Context, jid string, deviceID string) ([]map[string]interface{}, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}
	// Status updates are received via events/webhooks
	// This function returns a placeholder - actual status updates come via Message events
	// with Chat = status@broadcast
	_ = client
	return []map[string]interface{}{
		{
			"note": "Status updates are delivered via webhook events with event_type 'message.received' where chat is 'status@broadcast'",
		},
	}, nil
}

func WhatsAppDeleteStatus(ctx context.Context, jid string, deviceID string, statusID string) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}
	statusJID := types.StatusBroadcastJID
	// Delete the status message using revoke
	revokeMsg := client.BuildRevoke(statusJID, types.EmptyJID, statusID)
	_, err = client.SendMessage(ctx, statusJID, revokeMsg)
	return err
}

func WhatsAppGetUserStatus(ctx context.Context, jid string, deviceID string, userJID string) ([]map[string]interface{}, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}
	// User statuses are received via events
	// This provides the user's "about" status text
	parsedJID, err := types.ParseJID(userJID)
	if err != nil {
		parsedJID, err = WhatsAppCheckJID(ctx, jid, deviceID, userJID)
		if err != nil {
			return nil, err
		}
	}
	userInfo, err := client.GetUserInfo(ctx, []types.JID{parsedJID})
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0)
	for userJIDKey, info := range userInfo {
		result = append(result, map[string]interface{}{
			"jid":            userJIDKey.String(),
			"status":         info.Status,
			"verified_name":  info.VerifiedName,
			"picture_id":     info.PictureID,
		})
	}
	return result, nil
}

// Newsletter/Channel functions

func WhatsAppGetSubscribedNewsletters(ctx context.Context, jid string, deviceID string) ([]map[string]interface{}, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}
	newsletters, err := client.GetSubscribedNewsletters(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, len(newsletters))
	for i, n := range newsletters {
		result[i] = map[string]interface{}{
			"jid":              n.ID.String(),
			"name":             n.ThreadMeta.Name.Text,
			"description":      n.ThreadMeta.Description.Text,
			"subscriber_count": n.ThreadMeta.SubscriberCount,
			"verification":     n.ThreadMeta.VerificationState,
			"picture":          n.ThreadMeta.Picture,
			"preview":          n.ThreadMeta.Preview,
			"state":            n.State,
		}
	}
	return result, nil
}

func WhatsAppCreateNewsletter(ctx context.Context, jid string, deviceID string, name string, description string) (map[string]interface{}, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}
	params := whatsmeow.CreateNewsletterParams{
		Name:        name,
		Description: description,
	}
	newsletter, err := client.CreateNewsletter(ctx, params)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"jid":         newsletter.ID.String(),
		"name":        newsletter.ThreadMeta.Name.Text,
		"description": newsletter.ThreadMeta.Description.Text,
	}, nil
}

func WhatsAppGetNewsletterInfo(ctx context.Context, jid string, deviceID string, newsletterJID string) (map[string]interface{}, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}
	parsedJID, err := types.ParseJID(newsletterJID)
	if err != nil {
		return nil, fmt.Errorf("invalid newsletter JID: %w", err)
	}
	info, err := client.GetNewsletterInfo(ctx, parsedJID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"jid":              info.ID.String(),
		"name":             info.ThreadMeta.Name.Text,
		"description":      info.ThreadMeta.Description.Text,
		"subscriber_count": info.ThreadMeta.SubscriberCount,
		"verification":     info.ThreadMeta.VerificationState,
		"picture":          info.ThreadMeta.Picture,
		"preview":          info.ThreadMeta.Preview,
		"state":            info.State,
	}, nil
}

func WhatsAppFollowNewsletter(ctx context.Context, jid string, deviceID string, newsletterJID string) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}
	parsedJID, err := types.ParseJID(newsletterJID)
	if err != nil {
		return fmt.Errorf("invalid newsletter JID: %w", err)
	}
	return client.FollowNewsletter(ctx, parsedJID)
}

func WhatsAppUnfollowNewsletter(ctx context.Context, jid string, deviceID string, newsletterJID string) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}
	parsedJID, err := types.ParseJID(newsletterJID)
	if err != nil {
		return fmt.Errorf("invalid newsletter JID: %w", err)
	}
	return client.UnfollowNewsletter(ctx, parsedJID)
}

func WhatsAppGetNewsletterMessages(ctx context.Context, jid string, deviceID string, newsletterJID string, count int, before int) ([]map[string]interface{}, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}
	parsedJID, err := types.ParseJID(newsletterJID)
	if err != nil {
		return nil, fmt.Errorf("invalid newsletter JID: %w", err)
	}
	params := &whatsmeow.GetNewsletterMessagesParams{
		Count: count,
	}
	if before > 0 {
		params.Before = before
	}
	messages, err := client.GetNewsletterMessages(ctx, parsedJID, params)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, len(messages))
	for i, m := range messages {
		result[i] = map[string]interface{}{
			"server_id":  m.MessageServerID,
			"views":      m.ViewsCount,
			"reactions":  m.ReactionCounts,
		}
	}
	return result, nil
}

func WhatsAppSendNewsletterMessage(ctx context.Context, jid string, deviceID string, newsletterJID string, text string) (string, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}
	parsedJID, err := types.ParseJID(newsletterJID)
	if err != nil {
		return "", fmt.Errorf("invalid newsletter JID: %w", err)
	}
	if err := waitRateLimit(ctx, deviceID); err != nil {
		return "", err
	}
	msgContent := &waE2E.Message{
		Conversation: proto.String(text),
	}
	resp, err := client.SendMessage(ctx, parsedJID, msgContent)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func WhatsAppNewsletterSendReaction(ctx context.Context, jid string, deviceID string, newsletterJID string, messageServerID int, emoji string) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}
	parsedJID, err := types.ParseJID(newsletterJID)
	if err != nil {
		return fmt.Errorf("invalid newsletter JID: %w", err)
	}
	return client.NewsletterSendReaction(ctx, parsedJID, types.MessageServerID(messageServerID), emoji, "")
}

func WhatsAppNewsletterToggleMute(ctx context.Context, jid string, deviceID string, newsletterJID string, mute bool) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}
	parsedJID, err := types.ParseJID(newsletterJID)
	if err != nil {
		return fmt.Errorf("invalid newsletter JID: %w", err)
	}
	return client.NewsletterToggleMute(ctx, parsedJID, mute)
}

func WhatsAppNewsletterMarkViewed(ctx context.Context, jid string, deviceID string, newsletterJID string, messageServerIDs []int) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}
	parsedJID, err := types.ParseJID(newsletterJID)
	if err != nil {
		return fmt.Errorf("invalid newsletter JID: %w", err)
	}
	// Convert []int to []types.MessageServerID
	serverIDs := make([]types.MessageServerID, len(messageServerIDs))
	for i, id := range messageServerIDs {
		serverIDs[i] = types.MessageServerID(id)
	}
	return client.NewsletterMarkViewed(ctx, parsedJID, serverIDs)
}

func WhatsAppGetNewsletterInfoWithInvite(ctx context.Context, jid string, deviceID string, inviteCode string) (map[string]interface{}, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}
	info, err := client.GetNewsletterInfoWithInvite(ctx, inviteCode)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"jid":              info.ID.String(),
		"name":             info.ThreadMeta.Name.Text,
		"description":      info.ThreadMeta.Description.Text,
		"subscriber_count": info.ThreadMeta.SubscriberCount,
		"verification":     info.ThreadMeta.VerificationState,
		"picture":          info.ThreadMeta.Picture,
		"preview":          info.ThreadMeta.Preview,
		"state":            info.State,
	}, nil
}

func WhatsAppNewsletterSubscribeLiveUpdates(ctx context.Context, jid string, deviceID string, newsletterJID string) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}
	parsedJID, err := types.ParseJID(newsletterJID)
	if err != nil {
		return fmt.Errorf("invalid newsletter JID: %w", err)
	}
	_, err = client.NewsletterSubscribeLiveUpdates(ctx, parsedJID)
	return err
}

func WhatsAppUploadNewsletterPhoto(ctx context.Context, jid string, deviceID string, newsletterJID string, photoBytes []byte) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}
	parsedJID, err := types.ParseJID(newsletterJID)
	if err != nil {
		return fmt.Errorf("invalid newsletter JID: %w", err)
	}
	_, err = client.UploadNewsletter(ctx, photoBytes, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("failed to upload newsletter photo: %w", err)
	}
	// Note: The actual setting of the photo requires additional API calls
	// that may vary based on whatsmeow version
	_ = parsedJID
	return nil
}

// Poll functions

func WhatsAppCreatePoll(ctx context.Context, jid string, deviceID string, rjid string, question string, options []string, multiAnswer bool) (string, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}
	if strings.TrimSpace(question) == "" {
		return "", errors.New("Poll question cannot be empty")
	}
	if len(options) < 2 {
		return "", errors.New("Poll must have at least 2 options")
	}
	if len(options) > 12 {
		return "", errors.New("Poll cannot have more than 12 options")
	}
	remoteJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, rjid)
	if err != nil {
		return "", err
	}
	if err := waitRateLimit(ctx, deviceID); err != nil {
		return "", err
	}
	selectableCount := 1
	if multiAnswer {
		selectableCount = len(options)
	}
	pollMsg := client.BuildPollCreation(question, options, selectableCount)
	msgExtra := whatsmeow.SendRequestExtra{ID: client.GenerateMessageID()}
	_, err = client.SendMessage(ctx, remoteJID, pollMsg, msgExtra)
	if err != nil {
		return "", err
	}
	return msgExtra.ID, nil
}

func WhatsAppVotePoll(ctx context.Context, jid string, deviceID string, rjid string, pollMsgID string, selectedOptions []string) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}
	if len(selectedOptions) == 0 {
		return errors.New("At least one option must be selected")
	}
	remoteJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, rjid)
	if err != nil {
		return err
	}
	if err := waitRateLimit(ctx, deviceID); err != nil {
		return err
	}
	// Build the poll vote message
	// Note: BuildPollVote requires the original poll message info
	// This is a simplified implementation that may need adjustment based on actual poll message storage
	pollMsgInfo := &types.MessageInfo{
		ID:        pollMsgID,
		MessageSource: types.MessageSource{
			Chat: remoteJID,
		},
	}
	pollVoteMsg, err := client.BuildPollVote(ctx, pollMsgInfo, selectedOptions)
	if err != nil {
		return fmt.Errorf("failed to build poll vote: %w", err)
	}
	_, err = client.SendMessage(ctx, remoteJID, pollVoteMsg)
	if err != nil {
		return err
	}
	return nil
}

func WhatsAppSendVideo(ctx context.Context, jid string, deviceID string, rjid string, videoBytes []byte, videoType string, videoCaption string, isViewOnce bool, opts *SendOptions) (string, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}
	if len(videoBytes) == 0 {
		return "", errors.New("Video payload cannot be empty")
	}
	if err := enforceSizeLimit("video", int64(len(videoBytes)), maxVideoBytes); err != nil {
		return "", err
	}
	videoMime := detectMime(videoBytes, videoType)
	allowedVideoMimes := map[string]bool{
		"video/mp4":       true,
		"video/3gpp":      true,
		"video/quicktime": true,
		"video/webm":      true,
		"video/mpeg":      true,
	}
	if !allowedVideoMimes[videoMime] {
		return "", fmt.Errorf("Video MIME type %s is not allowed", videoMime)
	}
	remoteJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, rjid)
	if err != nil {
		return "", err
	}
	if err := waitRateLimit(ctx, deviceID); err != nil {
		return "", err
	}
	cleanup := beginPresenceSimulation(ctx, jid, deviceID, remoteJID, false, opts)
	defer cleanup()
	videoUploaded, err := client.Upload(ctx, videoBytes, whatsmeow.MediaVideo)
	if err != nil {
		return "", errors.New("Error While Uploading Video to WhatsApp Server")
	}
	msgExtra := whatsmeow.SendRequestExtra{ID: client.GenerateMessageID()}
	msgContent := &waE2E.Message{
		VideoMessage: &waE2E.VideoMessage{
			URL:           proto.String(videoUploaded.URL),
			DirectPath:    proto.String(videoUploaded.DirectPath),
			Mimetype:      proto.String(videoMime),
			Caption:       proto.String(videoCaption),
			FileLength:    proto.Uint64(videoUploaded.FileLength),
			FileSHA256:    videoUploaded.FileSHA256,
			FileEncSHA256: videoUploaded.FileEncSHA256,
			MediaKey:      videoUploaded.MediaKey,
			ViewOnce:      proto.Bool(isViewOnce),
		},
	}
	_, err = client.SendMessage(ctx, remoteJID, msgContent, msgExtra)
	if err != nil {
		return "", err
	}
	return msgExtra.ID, nil
}

func WhatsAppSendAudio(ctx context.Context, jid string, deviceID string, rjid string, audioBytes []byte, audioType string, isVoiceNote bool, opts *SendOptions) (string, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}
	if len(audioBytes) == 0 {
		return "", errors.New("Audio payload cannot be empty")
	}
	if err := enforceSizeLimit("audio", int64(len(audioBytes)), maxAudioBytes); err != nil {
		return "", err
	}
	audioMime := detectMime(audioBytes, audioType)
	allowedAudioMimes := map[string]bool{
		"audio/mpeg":      true,
		"audio/mp3":       true,
		"audio/mp4":       true,
		"audio/ogg":       true,
		"audio/wav":       true,
		"audio/x-wav":     true,
		"audio/aac":       true,
		"audio/opus":      true,
		"audio/ogg; codecs=opus": true,
	}
	if !allowedAudioMimes[audioMime] {
		return "", fmt.Errorf("Audio MIME type %s is not allowed", audioMime)
	}
	remoteJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, rjid)
	if err != nil {
		return "", err
	}
	if err := waitRateLimit(ctx, deviceID); err != nil {
		return "", err
	}
	cleanup := beginPresenceSimulation(ctx, jid, deviceID, remoteJID, true, opts)
	defer cleanup()
	audioUploaded, err := client.Upload(ctx, audioBytes, whatsmeow.MediaAudio)
	if err != nil {
		return "", errors.New("Error While Uploading Audio to WhatsApp Server")
	}
	msgExtra := whatsmeow.SendRequestExtra{ID: client.GenerateMessageID()}
	msgContent := &waE2E.Message{
		AudioMessage: &waE2E.AudioMessage{
			URL:           proto.String(audioUploaded.URL),
			DirectPath:    proto.String(audioUploaded.DirectPath),
			Mimetype:      proto.String(audioMime),
			FileLength:    proto.Uint64(audioUploaded.FileLength),
			FileSHA256:    audioUploaded.FileSHA256,
			FileEncSHA256: audioUploaded.FileEncSHA256,
			MediaKey:      audioUploaded.MediaKey,
			PTT:           proto.Bool(isVoiceNote),
		},
	}
	_, err = client.SendMessage(ctx, remoteJID, msgContent, msgExtra)
	if err != nil {
		return "", err
	}
	return msgExtra.ID, nil
}

func WhatsAppSendSticker(ctx context.Context, jid string, deviceID string, rjid string, stickerBytes []byte, opts *SendOptions) (string, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}
	if len(stickerBytes) == 0 {
		return "", errors.New("Sticker payload cannot be empty")
	}
	if err := enforceSizeLimit("sticker", int64(len(stickerBytes)), maxImageBytes); err != nil {
		return "", err
	}
	stickerMime := detectMime(stickerBytes, "image/webp")
	allowedStickerMimes := map[string]bool{
		"image/webp": true,
	}
	if !allowedStickerMimes[stickerMime] {
		return "", fmt.Errorf("Sticker MIME type %s is not allowed. Stickers must be in WebP format", stickerMime)
	}
	remoteJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, rjid)
	if err != nil {
		return "", err
	}
	if err := waitRateLimit(ctx, deviceID); err != nil {
		return "", err
	}
	cleanup := beginPresenceSimulation(ctx, jid, deviceID, remoteJID, false, opts)
	defer cleanup()
	stickerUploaded, err := client.Upload(ctx, stickerBytes, whatsmeow.MediaImage)
	if err != nil {
		return "", errors.New("Error While Uploading Sticker to WhatsApp Server")
	}
	msgExtra := whatsmeow.SendRequestExtra{ID: client.GenerateMessageID()}
	msgContent := &waE2E.Message{
		StickerMessage: &waE2E.StickerMessage{
			URL:           proto.String(stickerUploaded.URL),
			DirectPath:    proto.String(stickerUploaded.DirectPath),
			Mimetype:      proto.String(stickerMime),
			FileLength:    proto.Uint64(stickerUploaded.FileLength),
			FileSHA256:    stickerUploaded.FileSHA256,
			FileEncSHA256: stickerUploaded.FileEncSHA256,
			MediaKey:      stickerUploaded.MediaKey,
		},
	}
	_, err = client.SendMessage(ctx, remoteJID, msgContent, msgExtra)
	if err != nil {
		return "", err
	}
	return msgExtra.ID, nil
}

func WhatsAppSendLocation(ctx context.Context, jid string, deviceID string, rjid string, latitude float64, longitude float64, name string, address string, opts *SendOptions) (string, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}
	if latitude < -90 || latitude > 90 {
		return "", errors.New("Latitude must be between -90 and 90")
	}
	if longitude < -180 || longitude > 180 {
		return "", errors.New("Longitude must be between -180 and 180")
	}
	remoteJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, rjid)
	if err != nil {
		return "", err
	}
	if err := waitRateLimit(ctx, deviceID); err != nil {
		return "", err
	}
	cleanup := beginPresenceSimulation(ctx, jid, deviceID, remoteJID, false, opts)
	defer cleanup()
	msgExtra := whatsmeow.SendRequestExtra{ID: client.GenerateMessageID()}
	msgContent := &waE2E.Message{
		LocationMessage: &waE2E.LocationMessage{
			DegreesLatitude:  proto.Float64(latitude),
			DegreesLongitude: proto.Float64(longitude),
			Name:             proto.String(name),
			Address:          proto.String(address),
		},
	}
	_, err = client.SendMessage(ctx, remoteJID, msgContent, msgExtra)
	if err != nil {
		return "", err
	}
	return msgExtra.ID, nil
}

func WhatsAppSendContact(ctx context.Context, jid string, deviceID string, rjid string, contactName string, contactPhone string, opts *SendOptions) (string, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}
	if strings.TrimSpace(contactName) == "" {
		return "", errors.New("Contact name cannot be empty")
	}
	if strings.TrimSpace(contactPhone) == "" {
		return "", errors.New("Contact phone cannot be empty")
	}
	remoteJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, rjid)
	if err != nil {
		return "", err
	}
	if err := waitRateLimit(ctx, deviceID); err != nil {
		return "", err
	}
	cleanup := beginPresenceSimulation(ctx, jid, deviceID, remoteJID, false, opts)
	defer cleanup()
	// Build vCard format
	vcard := fmt.Sprintf("BEGIN:VCARD\nVERSION:3.0\nFN:%s\nTEL;type=CELL;type=VOICE;waid=%s:+%s\nEND:VCARD",
		contactName, contactPhone, contactPhone)
	msgExtra := whatsmeow.SendRequestExtra{ID: client.GenerateMessageID()}
	msgContent := &waE2E.Message{
		ContactMessage: &waE2E.ContactMessage{
			DisplayName: proto.String(contactName),
			Vcard:       proto.String(vcard),
		},
	}
	_, err = client.SendMessage(ctx, remoteJID, msgContent, msgExtra)
	if err != nil {
		return "", err
	}
	return msgExtra.ID, nil
}

func WhatsAppMessageEdit(ctx context.Context, jid string, deviceID string, rjid string, msgid string, message string) (string, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}
	if strings.TrimSpace(message) == "" {
		return "", errors.New("Message cannot be empty")
	}
	remoteJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, rjid)
	if err != nil {
		return "", err
	}
	WhatsAppPresence(context.Background(), jid, deviceID, true)
	WhatsAppComposeStatus(context.Background(), jid, deviceID, remoteJID, true, false)
	defer func() {
		WhatsAppComposeStatus(context.Background(), jid, deviceID, remoteJID, false, false)
		WhatsAppPresence(context.Background(), jid, deviceID, false)
	}()
	msgContent := &waE2E.Message{
		Conversation: proto.String(message),
	}
	_, err = client.SendMessage(ctx, remoteJID, client.BuildEdit(remoteJID, msgid, msgContent))
	if err != nil {
		return "", err
	}
	return msgid, nil
}

func WhatsAppMessageReact(ctx context.Context, jid string, deviceID string, rjid string, msgid string, emoji string) (string, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}
	remoteJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, rjid)
	if err != nil {
		return "", err
	}
	WhatsAppPresence(context.Background(), jid, deviceID, true)
	WhatsAppComposeStatus(context.Background(), jid, deviceID, remoteJID, true, false)
	defer func() {
		WhatsAppComposeStatus(context.Background(), jid, deviceID, remoteJID, false, false)
		WhatsAppPresence(context.Background(), jid, deviceID, false)
	}()
	if !gomoji.ContainsEmoji(emoji) && uniseg.GraphemeClusterCount(emoji) != 1 {
		return "", errors.New("WhatsApp Message React Emoji Must Be Contain Only 1 Emoji Character")
	}
	msgReact := &waE2E.Message{
		ReactionMessage: &waE2E.ReactionMessage{
			Key: &waCommon.MessageKey{
				FromMe:    proto.Bool(true),
				ID:        proto.String(msgid),
				RemoteJID: proto.String(remoteJID.String()),
			},
			Text:              proto.String(emoji),
			SenderTimestampMS: proto.Int64(time.Now().UnixMilli()),
		},
	}
	_, err = client.SendMessage(ctx, remoteJID, msgReact)
	if err != nil {
		return "", err
	}
	return msgid, nil
}

func WhatsAppPresenceStatus(ctx context.Context, jid string, deviceID string, status string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}

	presence := types.PresenceUnavailable
	if status == "available" {
		presence = types.PresenceAvailable
	}

	return client.SendPresence(ctx, presence)
}

func WhatsAppPresenceChat(ctx context.Context, jid string, deviceID string, chatID string, presenceType string, mediaType string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}

	remoteJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, chatID)
	if err != nil {
		return err
	}

	var chatPresence types.ChatPresence
	var mediaPresence types.ChatPresenceMedia

	switch presenceType {
	case "typing":
		chatPresence = types.ChatPresenceComposing
		mediaPresence = types.ChatPresenceMediaText
	case "recording":
		chatPresence = types.ChatPresenceComposing
		switch mediaType {
		case "video":
			mediaPresence = types.ChatPresenceMediaText
		default:
			mediaPresence = types.ChatPresenceMediaAudio
		}
	case "paused":
		chatPresence = types.ChatPresencePaused
		mediaPresence = types.ChatPresenceMediaText
	}

	return client.SendChatPresence(ctx, remoteJID, chatPresence, mediaPresence)
}

func WhatsAppGroupJoinWithInvite(jid string, deviceID string, groupID string, inviter string, code string, expiration int64) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}

	groupJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, groupID)
	if err != nil {
		return err
	}

	inviterJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, inviter)
	if err != nil {
		return err
	}

	err = client.JoinGroupWithInvite(context.Background(), groupJID, inviterJID, code, expiration)
	if err == nil {
		invalidateGroupListCache(deviceID)
	}
	return err
}

func WhatsAppGroupGetInfoFromInvite(jid string, deviceID string, groupID string, inviter string, code string, expiration int64) (*types.GroupInfo, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	groupJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, groupID)
	if err != nil {
		return nil, err
	}

	inviterJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, inviter)
	if err != nil {
		return nil, err
	}

	return client.GetGroupInfoFromInvite(context.Background(), groupJID, inviterJID, code, expiration)
}

func WhatsAppGroupGetInfoFromLink(ctx context.Context, jid string, deviceID string, code string) (*types.GroupInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	return client.GetGroupInfoFromLink(ctx, code)
}

func WhatsAppGroupSetMemberAddMode(jid string, deviceID string, gjid string, mode string) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}

	groupJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, gjid)
	if err != nil {
		return err
	}

	var memberAddMode types.GroupMemberAddMode
	switch mode {
	case "all_members":
		memberAddMode = "all_members"
	case "admin_only":
		memberAddMode = "admin_add"
	default:
		return errors.New("invalid member add mode")
	}

	err = client.SetGroupMemberAddMode(context.Background(), groupJID, memberAddMode)
	if err == nil {
		invalidateGroupListCache(deviceID)
	}
	return err
}

func WhatsAppGroupSetTopic(jid string, deviceID string, gjid string, previousID string, newID string, topic string) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}

	groupJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, gjid)
	if err != nil {
		return err
	}

	err = client.SetGroupTopic(context.Background(), groupJID, previousID, newID, topic)
	if err == nil {
		invalidateGroupListCache(deviceID)
	}
	return err
}

func WhatsAppGroupLink(jid string, deviceID string, parent string, child string) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}

	parentJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, parent)
	if err != nil {
		return err
	}

	childJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, child)
	if err != nil {
		return err
	}

	err = client.LinkGroup(context.Background(), parentJID, childJID)
	if err == nil {
		invalidateGroupListCache(deviceID)
	}
	return err
}

func WhatsAppGroupGetLinkedParticipants(ctx context.Context, jid string, deviceID string, community string) ([]types.JID, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	communityJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, community)
	if err != nil {
		return nil, err
	}

	return client.GetLinkedGroupsParticipants(ctx, communityJID)
}

func WhatsAppGroupGetSubGroups(ctx context.Context, jid string, deviceID string, community string) ([]*types.GroupLinkTarget, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	communityJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, community)
	if err != nil {
		return nil, err
	}

	return client.GetSubGroups(ctx, communityJID)
}

func WhatsAppMessageForward(ctx context.Context, jid string, deviceID string, messageContent *waE2E.Message, toChatID string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}

	toJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, toChatID)
	if err != nil {
		return "", err
	}

	if messageContent == nil {
		return "", fmt.Errorf("message content cannot be nil")
	}

	forwardedContent := &waE2E.Message{}

	if messageContent.Conversation != nil {
		forwardedContent.Conversation = proto.String(*messageContent.Conversation)
	} else if messageContent.ImageMessage != nil {
		forwardedContent.ImageMessage = messageContent.ImageMessage
	} else if messageContent.VideoMessage != nil {
		forwardedContent.VideoMessage = messageContent.VideoMessage
	} else if messageContent.AudioMessage != nil {
		forwardedContent.AudioMessage = messageContent.AudioMessage
	} else if messageContent.DocumentMessage != nil {
		forwardedContent.DocumentMessage = messageContent.DocumentMessage
	} else if messageContent.StickerMessage != nil {
		forwardedContent.StickerMessage = messageContent.StickerMessage
	} else if messageContent.ContactMessage != nil {
		forwardedContent.ContactMessage = messageContent.ContactMessage
	} else if messageContent.LocationMessage != nil {
		forwardedContent.LocationMessage = messageContent.LocationMessage
	} else if messageContent.ExtendedTextMessage != nil {
		forwardedContent.ExtendedTextMessage = messageContent.ExtendedTextMessage
	} else if messageContent.PollCreationMessage != nil {
		forwardedContent.PollCreationMessage = messageContent.PollCreationMessage
	} else {
		forwardedContent.Conversation = proto.String("📎 Forwarded message")
	}

	forwardingScore := uint32(1)

	var originalForwardingScore uint32
	if messageContent.ExtendedTextMessage != nil && messageContent.ExtendedTextMessage.ContextInfo != nil {
		if fs := messageContent.ExtendedTextMessage.ContextInfo.ForwardingScore; fs != nil && *fs > 0 {
			originalForwardingScore = *fs
		}
	} else if messageContent.ImageMessage != nil && messageContent.ImageMessage.ContextInfo != nil {
		if fs := messageContent.ImageMessage.ContextInfo.ForwardingScore; fs != nil && *fs > 0 {
			originalForwardingScore = *fs
		}
	} else if messageContent.VideoMessage != nil && messageContent.VideoMessage.ContextInfo != nil {
		if fs := messageContent.VideoMessage.ContextInfo.ForwardingScore; fs != nil && *fs > 0 {
			originalForwardingScore = *fs
		}
	}

	if originalForwardingScore > 0 {
		forwardingScore = originalForwardingScore + 1
	}

	contextInfo := &waE2E.ContextInfo{
		IsForwarded:     proto.Bool(true),
		ForwardingScore: proto.Uint32(forwardingScore),
	}

	if forwardedContent.Conversation != nil {
		forwardedContent.ExtendedTextMessage = &waE2E.ExtendedTextMessage{
			Text:        forwardedContent.Conversation,
			ContextInfo: contextInfo,
		}
		forwardedContent.Conversation = nil
	} else if forwardedContent.ImageMessage != nil {
		forwardedContent.ImageMessage.ContextInfo = contextInfo
	} else if forwardedContent.VideoMessage != nil {
		forwardedContent.VideoMessage.ContextInfo = contextInfo
	} else if forwardedContent.AudioMessage != nil {
		forwardedContent.AudioMessage.ContextInfo = contextInfo
	} else if forwardedContent.DocumentMessage != nil {
		forwardedContent.DocumentMessage.ContextInfo = contextInfo
	} else if forwardedContent.StickerMessage != nil {
		forwardedContent.StickerMessage.ContextInfo = contextInfo
	} else if forwardedContent.ContactMessage != nil {
		forwardedContent.ContactMessage.ContextInfo = contextInfo
	} else if forwardedContent.LocationMessage != nil {
		forwardedContent.LocationMessage.ContextInfo = contextInfo
	} else if forwardedContent.ExtendedTextMessage != nil {
		forwardedContent.ExtendedTextMessage.ContextInfo = contextInfo
	} else if forwardedContent.PollCreationMessage != nil {
		forwardedContent.PollCreationMessage.ContextInfo = contextInfo
	}

	msgExtra := whatsmeow.SendRequestExtra{ID: client.GenerateMessageID()}
	resp, err := client.SendMessage(ctx, toJID, forwardedContent, msgExtra)
	if err != nil {
		return "", fmt.Errorf("failed to send forwarded message: %w", err)
	}

	return resp.ID, nil
}

func WhatsAppSetUserStatus(jid string, deviceID string, status string) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}
	return client.SetStatusMessage(context.Background(), status)
}

func WhatsAppGetUserPrivacy(jid string, deviceID string) (types.PrivacySettings, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return types.PrivacySettings{}, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return types.PrivacySettings{}, err
	}
	return client.GetPrivacySettings(context.Background()), nil
}

func WhatsAppSetUserPrivacy(ctx context.Context, jid string, deviceID string, setting string, value string) (types.PrivacySettings, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return types.PrivacySettings{}, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return types.PrivacySettings{}, err
	}

	var privacyType types.PrivacySettingType
	var privacyValue types.PrivacySetting

	switch setting {
	case "group_add":
		privacyType = types.PrivacySettingTypeGroupAdd
	case "last_seen":
		privacyType = types.PrivacySettingTypeLastSeen
	case "status":
		privacyType = types.PrivacySettingTypeStatus
	case "profile":
		privacyType = types.PrivacySettingTypeProfile
	case "read_receipts":
		privacyType = types.PrivacySettingTypeReadReceipts
	default:
		return types.PrivacySettings{}, fmt.Errorf("invalid privacy setting: %s", setting)
	}

	switch value {
	case "all":
		privacyValue = types.PrivacySettingAll
	case "contacts":
		privacyValue = types.PrivacySettingContacts
	case "contact_blacklist":
		privacyValue = types.PrivacySettingContactBlacklist
	case "none":
		privacyValue = types.PrivacySettingNone
	case "matched":
		privacyValue = types.PrivacySettingMatchLastSeen
	default:
		if setting == "read_receipts" {
			if value == "true" {
				privacyValue = types.PrivacySettingAll
			} else {
				privacyValue = types.PrivacySettingNone
			}
		} else {
			return types.PrivacySettings{}, fmt.Errorf("invalid privacy value: %s", value)
		}
	}

	return client.SetPrivacySetting(ctx, privacyType, privacyValue)
}

func WhatsAppGetUserInfo(ctx context.Context, jid string, deviceID string, jids []string) (map[string]types.UserInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	jidList := make([]types.JID, len(jids))
	for i, j := range jids {
		jidList[i], err = types.ParseJID(j)
		if err != nil {
			return nil, fmt.Errorf("invalid JID format: %s", j)
		}
	}

	result, err := client.GetUserInfo(ctx, jidList)
	if err != nil {
		return nil, err
	}

	stringMap := make(map[string]types.UserInfo)
	for k, v := range result {
		stringMap[k.String()] = v
	}
	return stringMap, nil
}

func WhatsAppGetUserProfilePicture(ctx context.Context, jid string, deviceID string, targetJID string, preview bool) (*types.ProfilePictureInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	target, err := types.ParseJID(targetJID)
	if err != nil {
		return nil, fmt.Errorf("invalid JID format: %s", targetJID)
	}

	params := &whatsmeow.GetProfilePictureParams{
		Preview: preview,
	}

	return client.GetProfilePictureInfo(ctx, target, params)
}

func WhatsAppBlockUser(ctx context.Context, jid string, deviceID string, targetJID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}

	target, err := types.ParseJID(targetJID)
	if err != nil {
		return fmt.Errorf("invalid JID format: %s", targetJID)
	}

	_, err = client.UpdateBlocklist(ctx, target, "block")
	return err
}

func WhatsAppUnblockUser(ctx context.Context, jid string, deviceID string, targetJID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}

	target, err := types.ParseJID(targetJID)
	if err != nil {
		return fmt.Errorf("invalid JID format: %s", targetJID)
	}

	_, err = client.UpdateBlocklist(ctx, target, "unblock")
	return err
}

func WhatsAppSetDisappearingTimer(ctx context.Context, jid string, deviceID string, timer int, chatJID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}

	duration := time.Duration(timer) * time.Second

	if chatJID == "" {
		return client.SetDefaultDisappearingTimer(ctx, duration)
	}

	chat, err := types.ParseJID(chatJID)
	if err != nil {
		return fmt.Errorf("invalid chat JID format: %s", chatJID)
	}

	return client.SetDisappearingTimer(ctx, chat, duration, time.Now())
}

func WhatsAppGetStatusPrivacy(ctx context.Context, jid string, deviceID string) ([]types.StatusPrivacy, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}
	return client.GetStatusPrivacy(ctx)
}

func WhatsAppAddParticipants(ctx context.Context, jid string, deviceID string, groupID string, participants []string) ([]types.GroupParticipant, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	groupJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, groupID)
	if err != nil {
		return nil, err
	}
	if groupJID.Server != types.GroupServer {
		return nil, ErrInvalidGroupID
	}

	jidList := make([]types.JID, 0, len(participants))
	for _, participant := range participants {
		parsed, err := WhatsAppCheckJID(context.Background(), jid, deviceID, participant)
		if err != nil {
			continue
		}
		if parsed.Server == types.GroupServer {
			continue
		}
		jidList = append(jidList, parsed)
	}

	updated, err := client.UpdateGroupParticipants(ctx, groupJID, jidList, whatsmeow.ParticipantChangeAdd)
	if err == nil {
		invalidateGroupListCache(deviceID)
	}
	return updated, err
}

func WhatsAppRemoveParticipants(ctx context.Context, jid string, deviceID string, groupID string, participants []string) ([]types.GroupParticipant, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	groupJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, groupID)
	if err != nil {
		return nil, err
	}
	if groupJID.Server != types.GroupServer {
		return nil, ErrInvalidGroupID
	}

	jidList := make([]types.JID, 0, len(participants))
	for _, participant := range participants {
		parsed, err := WhatsAppCheckJID(context.Background(), jid, deviceID, participant)
		if err != nil {
			continue
		}
		jidList = append(jidList, parsed)
	}

	updated, err := client.UpdateGroupParticipants(ctx, groupJID, jidList, whatsmeow.ParticipantChangeRemove)
	if err == nil {
		invalidateGroupListCache(deviceID)
	}
	return updated, err
}

func WhatsAppApproveJoinRequests(ctx context.Context, jid string, deviceID string, groupID string, userIDs []string) ([]types.GroupParticipant, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	groupJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, groupID)
	if err != nil {
		return nil, err
	}
	if groupJID.Server != types.GroupServer {
		return nil, ErrInvalidGroupID
	}

	jidList := make([]types.JID, 0, len(userIDs))
	for _, userID := range userIDs {
		parsed, err := WhatsAppCheckJID(context.Background(), jid, deviceID, userID)
		if err != nil {
			continue
		}
		jidList = append(jidList, parsed)
	}

	updated, err := client.UpdateGroupRequestParticipants(ctx, groupJID, jidList, whatsmeow.ParticipantChangeApprove)
	if err == nil {
		invalidateGroupListCache(deviceID)
	}
	return updated, err
}

func WhatsAppRejectJoinRequests(ctx context.Context, jid string, deviceID string, groupID string, userIDs []string) ([]types.GroupParticipant, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	groupJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, groupID)
	if err != nil {
		return nil, err
	}
	if groupJID.Server != types.GroupServer {
		return nil, ErrInvalidGroupID
	}

	jidList := make([]types.JID, 0, len(userIDs))
	for _, userID := range userIDs {
		parsed, err := WhatsAppCheckJID(context.Background(), jid, deviceID, userID)
		if err != nil {
			continue
		}
		jidList = append(jidList, parsed)
	}

	updated, err := client.UpdateGroupRequestParticipants(ctx, groupJID, jidList, whatsmeow.ParticipantChangeReject)
	if err == nil {
		invalidateGroupListCache(deviceID)
	}
	return updated, err
}

func WhatsAppPromoteAdmins(ctx context.Context, jid string, deviceID string, groupID string, userIDs []string) ([]types.GroupParticipant, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	groupJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, groupID)
	if err != nil {
		return nil, err
	}
	if groupJID.Server != types.GroupServer {
		return nil, ErrInvalidGroupID
	}

	jidList := make([]types.JID, 0, len(userIDs))
	for _, userID := range userIDs {
		parsed, err := WhatsAppCheckJID(context.Background(), jid, deviceID, userID)
		if err != nil {
			continue
		}
		jidList = append(jidList, parsed)
	}

	updated, err := client.UpdateGroupParticipants(ctx, groupJID, jidList, whatsmeow.ParticipantChangePromote)
	if err == nil {
		invalidateGroupListCache(deviceID)
	}
	return updated, err
}

func WhatsAppDemoteAdmins(ctx context.Context, jid string, deviceID string, groupID string, userIDs []string) ([]types.GroupParticipant, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	groupJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, groupID)
	if err != nil {
		return nil, err
	}
	if groupJID.Server != types.GroupServer {
		return nil, ErrInvalidGroupID
	}

	jidList := make([]types.JID, 0, len(userIDs))
	for _, userID := range userIDs {
		parsed, err := WhatsAppCheckJID(context.Background(), jid, deviceID, userID)
		if err != nil {
			continue
		}
		jidList = append(jidList, parsed)
	}

	updated, err := client.UpdateGroupParticipants(ctx, groupJID, jidList, whatsmeow.ParticipantChangeDemote)
	if err == nil {
		invalidateGroupListCache(deviceID)
	}
	return updated, err
}

func WhatsAppUpdateGroupSettings(jid string, deviceID string, groupID string, announce *bool, locked *bool, memberAddMode string, joinApproval *bool) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}

	groupJID, err := WhatsAppCheckJID(context.Background(), jid, deviceID, groupID)
	if err != nil {
		return err
	}
	if groupJID.Server != types.GroupServer {
		return ErrInvalidGroupID
	}

	if announce != nil {
		if err := client.SetGroupAnnounce(context.Background(), groupJID, *announce); err != nil {
			return err
		}
	}

	if locked != nil {
		if err := client.SetGroupLocked(context.Background(), groupJID, *locked); err != nil {
			return err
		}
	}

	if memberAddMode != "" {
		var mode types.GroupMemberAddMode
		switch memberAddMode {
		case "all":
			mode = types.GroupMemberAddModeAllMember
		case "admin_only":
			mode = types.GroupMemberAddModeAdmin
		default:
			return fmt.Errorf("invalid member_add_mode: %s", memberAddMode)
		}
		if err := client.SetGroupMemberAddMode(context.Background(), groupJID, mode); err != nil {
			return err
		}
	}

	if joinApproval != nil {
		if err := client.SetGroupJoinApprovalMode(context.Background(), groupJID, *joinApproval); err != nil {
			return err
		}
	}

	invalidateGroupListCache(deviceID)
	return nil
}

func WhatsAppCreateGroupEnhanced(ctx context.Context, jid string, deviceID string, name string, participants []string, description string, photoBase64 string) (*types.GroupInfo, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	req := whatsmeow.ReqCreateGroup{
		Name: name,
	}

	if len(participants) > 0 {
		jidList := make([]types.JID, 0, len(participants))
		for _, participant := range participants {
			parsed, err := WhatsAppCheckJID(context.Background(), jid, deviceID, participant)
			if err != nil {
				continue
			}
			if parsed.Server == types.GroupServer {
				continue
			}
			jidList = append(jidList, parsed)
		}
		req.Participants = jidList
	}

	group, err := client.CreateGroup(ctx, req)
	if err != nil {
		return nil, err
	}

	groupJID := group.JID

	if description != "" {
		if err := client.SetGroupDescription(ctx, groupJID, description); err != nil {
			return group, err
		}
	}

	if photoBase64 != "" {
		photoBytes, err := base64.StdEncoding.DecodeString(photoBase64)
		if err != nil {
			return group, fmt.Errorf("invalid base64 photo data: %v", err)
		}
		if _, err := client.SetGroupPhoto(ctx, groupJID, photoBytes); err != nil {
			return group, err
		}
	}
	invalidateGroupListCache(deviceID)
	return group, nil
}

func WhatsAppGetPrivacy(ctx context.Context, jid string, deviceID string) (types.PrivacySettings, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return types.PrivacySettings{}, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return types.PrivacySettings{}, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	privacy, err := client.TryFetchPrivacySettings(ctx, false)
	if err != nil {
		return types.PrivacySettings{}, err
	}
	if privacy == nil {
		return types.PrivacySettings{}, errors.New("privacy settings not available")
	}
	return *privacy, nil
}

func WhatsAppSetPrivacy(ctx context.Context, jid string, deviceID string, settingType types.PrivacySettingType, value types.PrivacySetting) (types.PrivacySettings, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return types.PrivacySettings{}, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return types.PrivacySettings{}, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return client.SetPrivacySetting(ctx, settingType, value)
}

func WhatsAppGetUserDevices(ctx context.Context, jid string, deviceID string, userJID types.JID) ([]types.JID, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}
	return client.GetUserDevicesContext(ctx, []types.JID{userJID})
}

func WhatsAppGetChatHistory(jid string, deviceID string, chatJID types.JID, limit int, before string, after string) (interface{}, error) {
	return map[string]interface{}{
		"chat_jid": chatJID.String(),
		"limit":    limit,
		"before":   before,
		"after":    after,
		"messages": []interface{}{},
	}, nil
}

func WhatsAppArchiveChat(ctx context.Context, jid string, deviceID string, chatJID types.JID, archive bool) error {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}
	return client.SendAppState(ctx, appstate.BuildMute(chatJID, archive, 0))
}

func WhatsAppPinChat(ctx context.Context, jid string, deviceID string, chatJID types.JID, pin bool) error {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}
	return client.SendAppState(ctx, appstate.BuildPin(chatJID, pin))
}

func WhatsAppMarkRead(jid string, deviceID string, chatJID types.JID, senderJID types.JID, messageID string) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}
	msgIDs := []types.MessageID{types.MessageID(messageID)}
	return client.MarkRead(context.Background(), msgIDs, time.Now(), chatJID, senderJID)
}

func WhatsAppReact(ctx context.Context, jid string, deviceID string, chatJID types.JID, senderJID types.JID, messageID string, emoji string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}

	msg := client.BuildReaction(chatJID, senderJID, messageID, emoji)
	resp, err := client.SendMessage(ctx, chatJID, msg)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func WhatsAppEditMessage(ctx context.Context, jid string, deviceID string, chatJID types.JID, messageID string, newText string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}

	msg := client.BuildEdit(chatJID, messageID, &waE2E.Message{
		Conversation: &newText,
	})
	resp, err := client.SendMessage(ctx, chatJID, msg)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func WhatsAppDeleteMessage(ctx context.Context, jid string, deviceID string, chatJID types.JID, senderJID types.JID, messageID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}

	msg := client.BuildRevoke(chatJID, senderJID, messageID)
	_, err = client.SendMessage(ctx, chatJID, msg)
	return err
}

func WhatsAppForwardMessage(jid string, deviceID string, messageID string, toChatJID types.JID) (string, error) {
	return "", errors.New("forward message requires stored message data")
}

func WhatsAppGetMessageThumbnail(jid string, deviceID string, messageID string) ([]byte, string, error) {
	return nil, "image/jpeg", errors.New("thumbnail download requires stored message data")
}

func WhatsAppGroupInfo(ctx context.Context, jid string, deviceID string, groupJID types.JID) (*types.GroupInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}
	return client.GetGroupInfo(ctx, groupJID)
}

func WhatsAppGroupUpdateName(ctx context.Context, jid string, deviceID string, groupJID types.JID, name string) error {
	return WhatsAppGroupSetName(ctx, jid, deviceID, groupJID.String(), name)
}

func WhatsAppGroupUpdateDescription(ctx context.Context, jid string, deviceID string, groupJID types.JID, description string) error {
	return WhatsAppGroupSetDescription(ctx, jid, deviceID, groupJID.String(), description)
}

func WhatsAppGroupUpdatePhoto(ctx context.Context, jid string, deviceID string, groupJID types.JID, photoFile multipart.File) (string, error) {
	buffer := bytes.NewBuffer(nil)
	_, err := io.Copy(buffer, photoFile)
	if err != nil {
		return "", err
	}
	return WhatsAppGroupSetPhoto(ctx, jid, deviceID, groupJID.String(), buffer.Bytes())
}

func WhatsAppGroupUpdateSettings(jid string, deviceID string, groupJID types.JID, req interface{}) error {
	return nil
}

func WhatsAppGroupParticipantRequests(ctx context.Context, jid string, deviceID string, groupJID types.JID) ([]types.GroupParticipantRequest, error) {
	return WhatsAppGroupGetRequestParticipants(ctx, jid, deviceID, groupJID.String())
}

func WhatsAppGroupJoinApprovalMode(jid string, deviceID string, groupJID types.JID, mode bool) error {
	return WhatsAppGroupSetJoinApprovalMode(jid, deviceID, groupJID.String(), mode)
}

func WhatsAppGroupInfoFromInvite(ctx context.Context, jid string, deviceID string, inviteCode string) (*types.GroupInfo, error) {
	return WhatsAppGroupGetInfoFromLink(ctx, jid, deviceID, inviteCode)
}

// WhatsAppFetchAppState fetches application state patches for synchronization
func WhatsAppFetchAppState(jid string, deviceID string, name string, fullSync bool, onlyIfNotSynced bool) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}

	patchName := appstate.WAPatchName(name)
	return client.FetchAppState(context.Background(), patchName, fullSync, onlyIfNotSynced)
}

// WhatsAppSendAppState sends application state synchronization patches
func WhatsAppSendAppState(jid string, deviceID string, patchInfo appstate.PatchInfo) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}

	return client.SendAppState(context.Background(), patchInfo)
}

// WhatsAppMarkNotDirty marks application state as clean to avoid unnecessary syncing
func WhatsAppMarkNotDirty(jid string, deviceID string, cleanType string, timestamp time.Time) error {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}

	return client.MarkNotDirty(context.Background(), cleanType, timestamp)
}

// DownloadableMessage is an interface for messages that can be downloaded
type DownloadableMessage interface {
	GetURL() string
	GetDirectPath() string
	GetMediaKey() []byte
	GetFileEncSHA256() []byte
	GetFileSHA256() []byte
	GetFileLength() uint64
}

// WhatsAppDownloadMedia downloads media from a message using direct URL or encrypted media key
func WhatsAppDownloadMedia(ctx context.Context, jid string, deviceID string, msg DownloadableMessage) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	return client.Download(ctx, msg)
}

// WhatsAppDownloadMediaWithURL downloads media using a direct URL (for thumbnails, profile pics, etc.)
func WhatsAppDownloadMediaWithURL(ctx context.Context, jid string, deviceID string, directPath string, encFileHash []byte, fileHash []byte, mediaKey []byte, fileLength int, mediaType whatsmeow.MediaType) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	return client.DownloadMediaWithPath(ctx, directPath, encFileHash, fileHash, mediaKey, fileLength, mediaType, "")
}

// WhatsAppDownloadThumbnail downloads a thumbnail from a message
func WhatsAppDownloadThumbnail(ctx context.Context, jid string, deviceID string, msg *waE2E.Message) ([]byte, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var thumbnail []byte
	var mimeType string = "image/jpeg"

	// Check various message types for thumbnails
	if img := msg.GetImageMessage(); img != nil {
		thumbnail = img.GetJPEGThumbnail()
	} else if vid := msg.GetVideoMessage(); vid != nil {
		thumbnail = vid.GetJPEGThumbnail()
	} else if sticker := msg.GetStickerMessage(); sticker != nil {
		thumbnail = sticker.GetPngThumbnail()
		mimeType = "image/png"
	} else if doc := msg.GetDocumentMessage(); doc != nil {
		thumbnail = doc.GetJPEGThumbnail()
	} else if link := msg.GetExtendedTextMessage(); link != nil {
		thumbnail = link.GetJPEGThumbnail()
	}

	if len(thumbnail) == 0 {
		return nil, "", errors.New("no thumbnail available for this message")
	}

	return thumbnail, mimeType, nil
}

// WhatsAppSetProfilePhoto sets the current user's profile photo
func WhatsAppSetProfilePhoto(ctx context.Context, jid string, deviceID string, photoBytes []byte) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}

	if len(photoBytes) == 0 {
		return "", errors.New("photo bytes cannot be empty")
	}

	// Set profile photo for the current user
	pictureID, err := client.SetGroupPhoto(ctx, types.EmptyJID, photoBytes)
	if err != nil {
		return "", fmt.Errorf("failed to set profile photo: %w", err)
	}

	return pictureID, nil
}

// WhatsAppContactSync checks which phone numbers are registered on WhatsApp
func WhatsAppContactSync(ctx context.Context, jid string, deviceID string, phones []string) ([]ContactSyncResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	if len(phones) == 0 {
		return nil, errors.New("phones list cannot be empty")
	}

	// Format phone numbers with + prefix if not present
	formattedPhones := make([]string, len(phones))
	for i, phone := range phones {
		if !strings.HasPrefix(phone, "+") {
			formattedPhones[i] = "+" + phone
		} else {
			formattedPhones[i] = phone
		}
	}

	results, err := client.IsOnWhatsApp(ctx, formattedPhones)
	if err != nil {
		return nil, fmt.Errorf("failed to check contacts: %w", err)
	}

	syncResults := make([]ContactSyncResult, len(results))
	for i, info := range results {
		verifiedName := ""
		if info.VerifiedName != nil && info.VerifiedName.Details != nil {
			verifiedName = info.VerifiedName.Details.GetVerifiedName()
		}
		syncResults[i] = ContactSyncResult{
			Phone:        formattedPhones[i],
			IsRegistered: info.IsIn,
			JID:          info.JID.String(),
			VerifiedName: verifiedName,
		}
	}

	return syncResults, nil
}

// ContactSyncResult represents the result of a contact sync check
type ContactSyncResult struct {
	Phone        string `json:"phone"`
	IsRegistered bool   `json:"is_registered"`
	JID          string `json:"jid,omitempty"`
	VerifiedName string `json:"verified_name,omitempty"`
}

// WhatsAppGetContacts retrieves all saved contacts
func WhatsAppGetContacts(ctx context.Context, jid string, deviceID string) (map[types.JID]types.ContactInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	return client.Store.Contacts.GetAllContacts(ctx)
}

// WhatsAppGetBlocklist retrieves the current user's blocklist
func WhatsAppGetBlocklist(ctx context.Context, jid string, deviceID string) ([]types.JID, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	blocklist, err := client.GetBlocklist(ctx)
	if err != nil {
		return nil, err
	}
	return blocklist.JIDs, nil
}

type filteredLogger struct {
	base waLog.Logger
}

const websocketEOFErrorMsg = "Error reading from websocket: failed to get reader: failed to read frame header: EOF"

func isWebsocketEOFError(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, strings.ToLower(websocketEOFErrorMsg)) ||
		(strings.Contains(lower, "error reading from websocket") && strings.Contains(lower, "failed to read frame header: eof"))
}

func newFilteredLogger(base waLog.Logger) waLog.Logger {
	return &filteredLogger{base: base}
}

func (l *filteredLogger) Errorf(msg string, args ...interface{}) {
	formatted := fmt.Sprintf(msg, args...)
	if isWebsocketEOFError(formatted) {
		l.base.Debugf("WebSocket closed after idle; auto-reconnecting soon: %s", formatted)
		return
	}
	l.base.Errorf(msg, args...)
}

func (l *filteredLogger) Warnf(msg string, args ...interface{}) {
	l.base.Warnf(msg, args...)
}

func (l *filteredLogger) Infof(msg string, args ...interface{}) {
	l.base.Infof(msg, args...)
}

func (l *filteredLogger) Debugf(msg string, args ...interface{}) {
	l.base.Debugf(msg, args...)
}

func (l *filteredLogger) Sub(module string) waLog.Logger {
	return newFilteredLogger(l.base.Sub(module))
}

// ============================================================
// NEW FEATURES - WhatsApp API Extensions
// ============================================================

// WhatsAppRejectCall rejects an incoming call
func WhatsAppRejectCall(ctx context.Context, jid string, deviceID string, callFrom string, callID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}

	fromJID, err := types.ParseJID(callFrom)
	if err != nil {
		return fmt.Errorf("invalid call from JID: %v", err)
	}

	return client.RejectCall(ctx, fromJID, callID)
}

// BusinessProfile represents a WhatsApp business profile
type BusinessProfile struct {
	JID             string                    `json:"jid"`
	Description     string                    `json:"description"`
	Address         string                    `json:"address"`
	Email           string                    `json:"email"`
	Categories      []BusinessProfileCategory `json:"categories"`
	ProfileOptions  map[string]string         `json:"profile_options"`
	Websites        []string                  `json:"websites"`
	BusinessHours   []BusinessProfileHours    `json:"business_hours"`
}

type BusinessProfileCategory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type BusinessProfileHours struct {
	DayOfWeek string `json:"day_of_week"`
	Mode      string `json:"mode"`
	OpenTime  string `json:"open_time"`
	CloseTime string `json:"close_time"`
}

// WhatsAppGetBusinessProfile retrieves a business profile for a given JID
func WhatsAppGetBusinessProfile(ctx context.Context, jid string, deviceID string, targetJID string) (*types.BusinessProfile, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	target, err := types.ParseJID(targetJID)
	if err != nil {
		return nil, fmt.Errorf("invalid target JID: %v", err)
	}

	return client.GetBusinessProfile(ctx, target)
}

// BusinessMessageLinkTarget represents a resolved business message link
type BusinessMessageLinkTarget struct {
	JID           string `json:"jid"`
	PushName      string `json:"push_name"`
	VerifiedName  string `json:"verified_name"`
	IsBusiness    bool   `json:"is_business"`
	Message       string `json:"message"`
}

// WhatsAppResolveBusinessMessageLink resolves a business message link (wa.me/message/XXX)
func WhatsAppResolveBusinessMessageLink(ctx context.Context, jid string, deviceID string, code string) (*types.BusinessMessageLinkTarget, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	return client.ResolveBusinessMessageLink(ctx, code)
}

// WhatsAppGetContactQRLink gets the current user's contact QR link
func WhatsAppGetContactQRLink(ctx context.Context, jid string, deviceID string, revoke bool) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}

	return client.GetContactQRLink(ctx, revoke)
}

// WhatsAppResolveContactQRLink resolves a contact QR link code
func WhatsAppResolveContactQRLink(ctx context.Context, jid string, deviceID string, code string) (*types.ContactQRLinkTarget, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	return client.ResolveContactQRLink(ctx, code)
}

// BotInfo represents basic info about a bot
type BotInfo struct {
	JID          string `json:"jid"`
	PluginType   string `json:"plugin_type"`
	PluginName   string `json:"plugin_name"`
}

// WhatsAppGetBotListV2 retrieves the list of available bots
func WhatsAppGetBotListV2(ctx context.Context, jid string, deviceID string) ([]types.BotListInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	return client.GetBotListV2(ctx)
}

// WhatsAppGetBotProfiles retrieves profiles for the given bots
func WhatsAppGetBotProfiles(ctx context.Context, jid string, deviceID string, botInfo []types.BotListInfo) ([]types.BotProfileInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	return client.GetBotProfiles(ctx, botInfo)
}

// WhatsAppSubscribePresence subscribes to presence updates for a user
func WhatsAppSubscribePresence(ctx context.Context, jid string, deviceID string, targetJID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}

	target, err := types.ParseJID(targetJID)
	if err != nil {
		return fmt.Errorf("invalid target JID: %v", err)
	}

	return client.SubscribePresence(ctx, target)
}

// WhatsAppGetNewsletterMessageUpdates retrieves message updates for a newsletter
func WhatsAppGetNewsletterMessageUpdates(ctx context.Context, jid string, deviceID string, newsletterJID string, count int, since int) ([]*types.NewsletterMessage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	nlJID, err := types.ParseJID(newsletterJID)
	if err != nil {
		return nil, fmt.Errorf("invalid newsletter JID: %v", err)
	}

	params := &whatsmeow.GetNewsletterUpdatesParams{
		Count: count,
		Since: time.Unix(int64(since), 0),
	}

	return client.GetNewsletterMessageUpdates(ctx, nlJID, params)
}

// WhatsAppAcceptTOSNotice accepts a Terms of Service notice (required for newsletter creation)
func WhatsAppAcceptTOSNotice(ctx context.Context, jid string, deviceID string, noticeID string, stage string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}

	return client.AcceptTOSNotice(ctx, noticeID, stage)
}

// WhatsAppSetPassive sets the client to passive mode
func WhatsAppSetPassive(ctx context.Context, jid string, deviceID string, passive bool) error {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}

	return client.SetPassive(ctx, passive)
}

// WhatsAppWaitForConnection waits for the client to be connected
func WhatsAppWaitForConnection(jid string, deviceID string, timeout time.Duration) bool {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return false
	}

	return client.WaitForConnection(timeout)
}

// WhatsAppSendMediaRetryReceipt sends a media retry receipt for failed media downloads
func WhatsAppSendMediaRetryReceipt(ctx context.Context, jid string, deviceID string, chatJID string, senderJID string, messageID string, mediaKey []byte) error {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}

	chat, err := types.ParseJID(chatJID)
	if err != nil {
		return fmt.Errorf("invalid chat JID: %v", err)
	}

	sender, err := types.ParseJID(senderJID)
	if err != nil {
		return fmt.Errorf("invalid sender JID: %v", err)
	}

	msgInfo := &types.MessageInfo{
		ID:        messageID,
		MessageSource: types.MessageSource{
			Chat:   chat,
			Sender: sender,
		},
	}

	return client.SendMediaRetryReceipt(ctx, msgInfo, mediaKey)
}

// WhatsAppBuildHistorySyncRequest builds a history sync request message
func WhatsAppBuildHistorySyncRequest(jid string, deviceID string, chatJID string, senderJID string, lastMsgID string, count int) (*waE2E.Message, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	chat, err := types.ParseJID(chatJID)
	if err != nil {
		return nil, fmt.Errorf("invalid chat JID: %v", err)
	}

	sender, err := types.ParseJID(senderJID)
	if err != nil {
		return nil, fmt.Errorf("invalid sender JID: %v", err)
	}

	msgInfo := &types.MessageInfo{
		ID:        lastMsgID,
		MessageSource: types.MessageSource{
			Chat:   chat,
			Sender: sender,
		},
	}

	return client.BuildHistorySyncRequest(msgInfo, count), nil
}

// WhatsAppBuildUnavailableMessageRequest builds a request for an unavailable message
func WhatsAppBuildUnavailableMessageRequest(jid string, deviceID string, chatJID string, senderJID string, messageID string) (*waE2E.Message, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	chat, err := types.ParseJID(chatJID)
	if err != nil {
		return nil, fmt.Errorf("invalid chat JID: %v", err)
	}

	sender, err := types.ParseJID(senderJID)
	if err != nil {
		return nil, fmt.Errorf("invalid sender JID: %v", err)
	}

	return client.BuildUnavailableMessageRequest(chat, sender, messageID), nil
}

// WhatsAppUnlinkGroup unlinks a child group from a parent community
func WhatsAppUnlinkGroup(ctx context.Context, jid string, deviceID string, parentJID string, childJID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}

	parent, err := types.ParseJID(parentJID)
	if err != nil {
		return fmt.Errorf("invalid parent JID: %v", err)
	}

	child, err := types.ParseJID(childJID)
	if err != nil {
		return fmt.Errorf("invalid child JID: %v", err)
	}

	return client.UnlinkGroup(ctx, parent, child)
}

// WhatsAppGenerateMessageID generates a new random message ID
func WhatsAppGenerateMessageID(jid string, deviceID string) (string, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return "", err
	}

	return string(client.GenerateMessageID()), nil
}

// WhatsAppUploadReader uploads media from a reader (streaming upload)
func WhatsAppUploadReader(ctx context.Context, jid string, deviceID string, reader io.Reader, tempFile io.ReadWriteSeeker, appInfo whatsmeow.MediaType) (*whatsmeow.UploadResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return nil, err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return nil, err
	}

	resp, err := client.UploadReader(ctx, reader, tempFile, appInfo)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// WhatsAppStoreLIDPNMapping stores a LID to phone number mapping
func WhatsAppStoreLIDPNMapping(ctx context.Context, jid string, deviceID string, firstJID string, secondJID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return err
	}
	if err = WhatsAppIsClientOK(jid, deviceID); err != nil {
		return err
	}

	first, err := types.ParseJID(firstJID)
	if err != nil {
		return fmt.Errorf("invalid first JID: %v", err)
	}

	second, err := types.ParseJID(secondJID)
	if err != nil {
		return fmt.Errorf("invalid second JID: %v", err)
	}

	client.StoreLIDPNMapping(ctx, first, second)
	return nil
}
