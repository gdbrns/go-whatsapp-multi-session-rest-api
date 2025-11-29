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
