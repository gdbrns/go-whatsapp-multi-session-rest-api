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
	EventConnectionConnected   EventType = "connection.connected"
	EventConnectionDisconnected EventType = "connection.disconnected"
	EventConnectionLoggedOut   EventType = "connection.logged_out"
	EventConnectionReconnecting EventType = "connection.reconnecting"
	EventConnectionKeepAliveTimeout EventType = "connection.keepalive_timeout"
	EventConnectionTemporaryBan EventType = "connection.temporary_ban"
	EventAppStateSyncComplete  EventType = "appstate.sync_complete"
	EventAppStatePatchReceived EventType = "appstate.patch_received"
	// Call events
	EventCallOffer             EventType = "call.offer"
	EventCallAccept            EventType = "call.accept"
	EventCallTerminate         EventType = "call.terminate"
	EventCallReject            EventType = "call.reject"
	// History sync events
	EventHistorySync           EventType = "history.sync"
	// Blocklist events
	EventBlocklistChange       EventType = "blocklist.change"
	// Group events
	EventGroupJoin             EventType = "group.join"
	EventGroupLeave            EventType = "group.leave"
	EventGroupParticipantUpdate EventType = "group.participant_update"
	EventGroupInfoUpdate       EventType = "group.info_update"
	// Contact events
	EventContactUpdate         EventType = "contact.update"
	// Newsletter/Channel events
	EventNewsletterJoin              EventType = "newsletter.join"
	EventNewsletterLeave             EventType = "newsletter.leave"
	EventNewsletterMessageReceived   EventType = "newsletter.message_received"
	EventNewsletterUpdate            EventType = "newsletter.update"
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
	EventHistorySyncComplete   EventType = "history.sync_complete"
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
