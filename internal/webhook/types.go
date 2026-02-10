package webhook

import (
	"time"
)

type EventType string

const (
	EventMessageReceived       EventType = "message.received"
	EventMessageDelivered      EventType = "message.delivered"
	EventMessageRead           EventType = "message.read"
	EventMessagePlayed         EventType = "message.played"
	EventMessageDeleted        EventType = "message.deleted"
	EventMessageUndecryptable  EventType = "message.undecryptable"
	EventMessageFBReceived     EventType = "message.fb_received"
	EventConnectionConnected   EventType = "connection.connected"
	EventConnectionDisconnected EventType = "connection.disconnected"
	EventConnectionLoggedOut   EventType = "connection.logged_out"
	EventConnectionReconnecting EventType = "connection.reconnecting"
	EventConnectionKeepAliveTimeout EventType = "connection.keepalive_timeout"
	EventConnectionKeepAliveRestored EventType = "connection.keepalive_restored"
	EventConnectionTemporaryBan EventType = "connection.temporary_ban"
	EventConnectionClientOutdated EventType = "connection.client_outdated"
	EventConnectionCATRefreshError EventType = "connection.cat_refresh_error"
	EventConnectionConnectFailure EventType = "connection.connect_failure"
	EventConnectionStreamError EventType = "connection.stream_error"
	EventConnectionStreamReplaced EventType = "connection.stream_replaced"
	EventConnectionManualLoginReconnect EventType = "connection.manual_login_reconnect"
	EventConnectionQR EventType = "connection.qr"
	EventConnectionQRScannedWithoutMultidevice EventType = "connection.qr_scanned_without_multidevice"
	EventConnectionPairSuccess EventType = "connection.pair_success"
	EventConnectionPairError EventType = "connection.pair_error"
	EventAppStateSyncComplete  EventType = "appstate.sync_complete"
	EventAppStateSyncError     EventType = "appstate.sync_error"
	EventAppStatePatchReceived EventType = "appstate.patch_received"
	// Call events
	EventCallOffer             EventType = "call.offer"
	EventCallAccept            EventType = "call.accept"
	EventCallPreAccept         EventType = "call.pre_accept"
	EventCallOfferNotice       EventType = "call.offer_notice"
	EventCallTransport         EventType = "call.transport"
	EventCallRelayLatency      EventType = "call.relay_latency"
	EventCallTerminate         EventType = "call.terminate"
	EventCallReject            EventType = "call.reject"
	EventCallUnknown           EventType = "call.unknown"
	// History sync events
	EventHistorySync           EventType = "history.sync"
	EventHistorySyncComplete   EventType = "history.sync_complete"
	EventOfflineSyncPreview    EventType = "offline.sync_preview"
	EventOfflineSyncCompleted  EventType = "offline.sync_completed"
	// Blocklist events
	EventBlocklistChange       EventType = "blocklist.change"
	// Group events
	EventGroupJoin             EventType = "group.join"
	EventGroupLeave            EventType = "group.leave"
	EventGroupParticipantUpdate EventType = "group.participant_update"
	EventGroupInfoUpdate       EventType = "group.info_update"
	// Contact events
	EventContactUpdate         EventType = "contact.update"
	EventChatPresence          EventType = "chat.presence"
	EventPresence              EventType = "presence.update"
	EventIdentityChange        EventType = "identity.change"
	EventPictureUpdate         EventType = "picture.update"
	EventUserAbout             EventType = "user.about"
	EventPrivacySettings       EventType = "privacy.settings"
	EventPushName              EventType = "pushname.update"
	EventPushNameSetting       EventType = "pushname.setting"
	EventBusinessName          EventType = "business.name_update"
	EventChatMute              EventType = "chat.mute"
	EventChatArchive           EventType = "chat.archive"
	EventChatPin               EventType = "chat.pin"
	EventChatStar              EventType = "chat.star"
	EventChatDeleteForMe       EventType = "chat.delete_for_me"
	EventChatDelete            EventType = "chat.delete"
	EventChatClear             EventType = "chat.clear"
	EventChatMarkRead          EventType = "chat.mark_read"
	EventLabelEdit             EventType = "label.edit"
	EventLabelAssociationChat  EventType = "label.chat"
	EventLabelAssociationMessage EventType = "label.message"
	EventUnarchiveChatsSetting EventType = "settings.unarchive_chats"
	EventUserStatusMute        EventType = "status.mute"
	// Newsletter/Channel events
	EventNewsletterJoin              EventType = "newsletter.join"
	EventNewsletterLeave             EventType = "newsletter.leave"
	EventNewsletterMessageReceived   EventType = "newsletter.message_received"
	EventNewsletterUpdate            EventType = "newsletter.update"
	EventNewsletterMuteChange        EventType = "newsletter.mute_change"
	EventNewsletterLiveUpdate        EventType = "newsletter.live_update"
	// Poll events
	EventPollCreated           EventType = "poll.created"
	EventPollVote              EventType = "poll.vote"
	EventPollUpdate            EventType = "poll.update"
	// Status/Stories events
	EventStatusPosted          EventType = "status.posted"
	EventStatusViewed          EventType = "status.viewed"
	EventStatusDeleted         EventType = "status.deleted"
	// Media events
	EventMediaReceived         EventType = "media.received"
	EventMediaDownloaded       EventType = "media.downloaded"
	// New event types for advanced features
	EventMediaRetry            EventType = "media.retry"
	EventPollVoteDecrypted     EventType = "poll.vote_decrypted"
	EventStatusComment         EventType = "status.comment"
)

type DeliveryStatus string

const (
	DeliveryPending   DeliveryStatus = "pending"
	DeliverySuccess   DeliveryStatus = "success"
	DeliveryFailed    DeliveryStatus = "failed"
	DeliveryRetrying  DeliveryStatus = "retrying"
)

type WebhookConfig struct {
	ID        int64
	DeviceID  string
	URL       string
	Secret    string
	Events    []EventType
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type WebhookEvent struct {
	EventType EventType              `json:"event_type"`
	DeviceID  string                 `json:"device_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

type DeliveryLog struct {
	ID           int64
	WebhookID    int64
	EventType    EventType
	Status       DeliveryStatus
	AttemptCount int
	LastError    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
