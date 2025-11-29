package whatsapp

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"runtime"
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

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/webhook"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/env"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/log"
)

type SessionKey struct {
	JID      string
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
	webhookEngine            *webhook.Engine
)

const (
	qrChannelWaitTimeout    = 2 * time.Minute
	pairPhoneRequestTimeout = 90 * time.Second
	logoutRequestTimeout    = 30 * time.Second
	routingCleanupTimeout   = 5 * time.Second
)

func init() {
	var err error

	dbType, err := env.GetEnvString("WHATSAPP_DATASTORE_TYPE")
	if err != nil {
		log.Print(nil).WithError(err).Fatal("Error parsing WHATSAPP_DATASTORE_TYPE")
	}

	dbURI, err := env.GetEnvString("WHATSAPP_DATASTORE_URI")
	if err != nil {
		log.Print(nil).WithError(err).Fatal("Error parsing WHATSAPP_DATASTORE_URI")
	}

	normalizedDriver := normalizeDatastoreDriver(dbType)
	dbURI = normalizeDatastoreDSN(normalizedDriver, dbURI)

	datastoreDriver = normalizedDriver
	datastoreDSN = dbURI

	log.Print(nil).Info("Initializing WhatsApp datastore with driver=" + normalizedDriver)

	datastore, err := sqlstore.New(context.Background(), normalizedDriver, dbURI, nil)
	if err != nil {
		log.Print(nil).WithError(err).Fatal("Failed to initialize WhatsApp client datastore")
	}

	WhatsAppClientProxyURL, _ = env.GetEnvString("WHATSAPP_CLIENT_PROXY_URL")

	if _, err := openRoutingDB(); err != nil {
		log.Print(nil).WithError(err).Fatal("Error initializing routing datastore")
	}

	WhatsAppDatastore = datastore

	if err := upgradeDatastoreSchema(context.Background()); err != nil {
		log.Print(nil).WithError(err).Fatal("Failed to upgrade datastore schema")
	}

	db, err := openRoutingDB()
	if err != nil {
		log.Print(nil).WithError(err).Fatal("Failed to open routing DB for webhooks")
	}
	webhookStore := webhook.NewStore(db)
	webhookEngine = webhook.NewEngine(webhookStore)

	log.Print(nil).Info("database is ok")
}

func upgradeDatastoreSchema(ctx context.Context) error {
	if WhatsAppDatastore == nil {
		return errors.New("whatsapp datastore not initialized")
	}

	if err := WhatsAppDatastore.Upgrade(ctx); err != nil {
		return fmt.Errorf("upgrade operation failed: %w", err)
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

func clientKey(jid string, deviceID string) SessionKey {
	return SessionKey{JID: jid, DeviceID: deviceID}
}

func getClient(jid string, deviceID string) *whatsmeow.Client {
	key := clientKey(jid, deviceID)
	clientsMu.RLock()
	client := WhatsAppClient[key]
	clientsMu.RUnlock()
	return client
}

func setClient(jid string, deviceID string, client *whatsmeow.Client) {
	key := clientKey(jid, deviceID)
	clientsMu.Lock()
	WhatsAppClient[key] = client
	clientsMu.Unlock()
}

func deleteClient(jid string, deviceID string) {
	key := clientKey(jid, deviceID)
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
		client := getClient(key.JID, key.DeviceID)
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

func WhatsAppClientsLen() int {
	return clientsLen()
}

func maskJIDForLog(jid string) string {
	if len(jid) < 4 {
		return jid
	}
	return jid[0:len(jid)-4] + "xxxx"
}

func WhatsAppRangeClients(fn func(jid string, deviceID string, client *whatsmeow.Client)) {
	rangeClients(func(key SessionKey, client *whatsmeow.Client) {
		fn(key.JID, key.DeviceID, client)
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

	if getClient(jid, deviceID) == nil {
		if device == nil {
			device = WhatsAppDatastore.NewDevice()
		}

		store.DeviceProps.Os = proto.String(runtime.GOOS)
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

		client := whatsmeow.NewClient(device, nil)

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
}

func handleWhatsAppEvents(jid string, deviceID string) func(interface{}) {
	return func(evt interface{}) {
		switch e := evt.(type) {
		case *events.LoggedOut:
			client, err := currentClient(jid, deviceID)
			if err == nil {
				client.Disconnect()
			}
			deleteClient(jid, deviceID)
			routingCtx, routingCancel := context.WithTimeout(context.Background(), routingCleanupTimeout)
			_ = DeleteDeviceRouting(routingCtx, deviceID)
			routingCancel()
			dispatchWebhook(deviceID, webhook.EventConnectionLoggedOut, map[string]interface{}{
				"jid": jid,
			})
		case *events.StreamReplaced:
			client, err := currentClient(jid, deviceID)
			if err == nil {
				client.Disconnect()
			}
			deleteClient(jid, deviceID)
			routingCtx, routingCancel := context.WithTimeout(context.Background(), routingCleanupTimeout)
			_ = DeleteDeviceRouting(routingCtx, deviceID)
			routingCancel()
		case *events.Connected:
			log.Print(nil).Info("Client connected: " + maskJIDForLog(jid) + " (" + deviceID + ")")
			client, err := currentClient(jid, deviceID)
			if err == nil && client.Store.ID != nil {
				_ = SaveDeviceRouting(context.Background(), deviceID, client.Store.ID.String())
			}
			dispatchWebhook(deviceID, webhook.EventConnectionConnected, map[string]interface{}{
				"jid": jid,
			})
		case *events.Disconnected:
			log.Print(nil).Warn("Client disconnected: " + maskJIDForLog(jid) + " (" + deviceID + ")")
			dispatchWebhook(deviceID, webhook.EventConnectionDisconnected, map[string]interface{}{
				"jid": jid,
			})
		case *events.Message:
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
		case *events.KeepAliveTimeout:
			log.Print(nil).Warn(fmt.Sprintf("Client keepalive timeout: %s (%s), errors=%d, lastSuccess=%s", maskJIDForLog(jid), deviceID, e.ErrorCount, e.LastSuccess.Format(time.RFC3339)))
		case *events.TemporaryBan:
			log.Print(nil).Error(fmt.Sprintf("Client temporarily banned: %s (%s), reason=%s, expires=%s", maskJIDForLog(jid), deviceID, e.Code, e.Expire))
		case *events.ConnectFailure:
			log.Print(nil).Error(fmt.Sprintf("Client connection failure: %s (%s), reason=%s, message=%s", maskJIDForLog(jid), deviceID, e.Reason, e.Message))
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

func WhatsAppLogin(jid string, deviceID string) (string, int, error) {
	client, err := currentClient(jid, deviceID)
	if err != nil {
		return "", 0, err
	}

	client.Disconnect()

	if client.Store.ID == nil {
		ctx, cancel := context.WithTimeout(context.Background(), qrChannelWaitTimeout)
		defer cancel()

		qrChanGenerate, err := client.GetQRChannel(ctx)
		if err != nil {
			return "", 0, err
		}

		err = client.Connect()
		if err != nil {
			return "", 0, err
		}

		qrImage, qrTimeout, paired, err := WhatsAppGenerateQR(ctx, qrChanGenerate)
		if err != nil {
			return "", 0, err
		}
		if paired {
			return "WhatsApp Client is already paired", 0, nil
		}

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
	infos, err := client.IsOnWhatsApp(ctx, []string{"+" + normalized})
	if err != nil {
		return types.EmptyJID
	}
	if len(infos) == 0 {
		return types.EmptyJID
	}
	if infos[0].IsIn {
		return infos[0].JID
	}
	return types.EmptyJID
}

func WhatsAppCheckJID(ctx context.Context, jid string, deviceID string, id string) (types.JID, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	_, err := currentClient(jid, deviceID)
	if err != nil {
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
	if parsed, err := types.ParseJID(WhatsAppDecomposeJID(id)); err == nil {
		return parsed
	}

	id = WhatsAppDecomposeJID(id)
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
	return client.LeaveGroup(ctx, groupJID)
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
	return client.SetGroupName(ctx, groupJID, name)
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
	return client.SetGroupDescription(ctx, groupJID, description)
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
	return client.SetGroupLocked(context.Background(), groupJID, locked)
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
	return client.SetGroupAnnounce(context.Background(), groupJID, announce)
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
	return client.SetGroupJoinApprovalMode(context.Background(), groupJID, mode)
}

func WhatsAppSendText(ctx context.Context, jid string, deviceID string, rjid string, message string) (string, error) {
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
	msgExtra := whatsmeow.SendRequestExtra{ID: client.GenerateMessageID()}
	msgContent := &waE2E.Message{
		Conversation: proto.String(message),
	}
	_, err = client.SendMessage(ctx, remoteJID, msgContent, msgExtra)
	if err != nil {
		return "", err
	}
	return msgExtra.ID, nil
}

func WhatsAppSendDocument(ctx context.Context, jid string, deviceID string, rjid string, documentBytes []byte, documentType string, documentName string) (string, error) {
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
	documentUploaded, err := client.Upload(ctx, documentBytes, whatsmeow.MediaDocument)
	if err != nil {
		return "", errors.New("Error While Uploading Media to WhatsApp Server")
	}
	msgExtra := whatsmeow.SendRequestExtra{ID: client.GenerateMessageID()}
	msgContent := &waE2E.Message{
		DocumentMessage: &waE2E.DocumentMessage{
			URL:           proto.String(documentUploaded.URL),
			DirectPath:    proto.String(documentUploaded.DirectPath),
			Mimetype:      proto.String(documentType),
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

func WhatsAppSendImage(ctx context.Context, jid string, deviceID string, rjid string, imageBytes []byte, imageType string, imageCaption string, isViewOnce bool) (string, error) {
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
			Mimetype:            proto.String(imageType),
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

func WhatsAppMessageEdit(ctx context.Context, jid string, deviceID string, rjid string, msgid string, message string) (string, error) {
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

	return client.JoinGroupWithInvite(context.Background(), groupJID, inviterJID, code, expiration)
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

	return client.SetGroupMemberAddMode(context.Background(), groupJID, memberAddMode)
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

	return client.SetGroupTopic(context.Background(), groupJID, previousID, newID, topic)
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

	return client.LinkGroup(context.Background(), parentJID, childJID)
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

	return client.UpdateGroupParticipants(ctx, groupJID, jidList, whatsmeow.ParticipantChangeAdd)
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

	return client.UpdateGroupParticipants(ctx, groupJID, jidList, whatsmeow.ParticipantChangeRemove)
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

	return client.UpdateGroupRequestParticipants(ctx, groupJID, jidList, whatsmeow.ParticipantChangeApprove)
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

	return client.UpdateGroupRequestParticipants(ctx, groupJID, jidList, whatsmeow.ParticipantChangeReject)
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

	return client.UpdateGroupParticipants(ctx, groupJID, jidList, whatsmeow.ParticipantChangePromote)
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

	return client.UpdateGroupParticipants(ctx, groupJID, jidList, whatsmeow.ParticipantChangeDemote)
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
