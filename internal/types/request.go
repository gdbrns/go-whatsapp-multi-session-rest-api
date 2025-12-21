package types

import waE2E "go.mau.fi/whatsmeow/proto/waE2E"

type RequestLogin struct {
	Output string
}

type RequestLoginCode struct {
	Phone string `json:"phone"`
}

type RequestCheckPhone struct {
	Phone string
}

type RequestUserInfo struct {
	JID   string
	Phone string
}

type RequestUserPicture struct {
	JID     string
	Phone   string
	Preview bool
}

type RequestBlockUser struct {
	JID string
}

type RequestUnblockUser struct {
	JID string
}

type RequestPrivacy struct {
	Setting string
	Value   string
}

type RequestStatus struct {
	Status string
}

type RequestSendMessage struct {
	Phone          string
	Message        string
	Text           string
	ReplyMessageID string
	ViewOnce       bool
	TypingSimulation   *bool `json:"typing_simulation"`
	PresenceSimulation *bool `json:"presence_simulation"`
}

type RequestSendLink struct {
	Phone          string
	Link           string
	Caption        string
	URL            string
	ReplyMessageID string
}

type RequestSendLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name"`
	Address   string  `json:"address"`
}

type RequestSendContact struct {
	Name  string
	Phone string
}

type RequestSendPoll struct {
	Question    string
	Options     []string
	MultiAnswer bool
}

type RequestSendPollVote struct {
	PollMessageID string
	Options       []string
}

type RequestSendImage struct {
	Caption  string
	ViewOnce bool
}

type RequestSendVideo struct {
	Caption  string
	ViewOnce bool
}

type RequestSendAudio struct {
	Caption string
}

type RequestSendDocument struct {
	Caption  string
	FileName string
}

type RequestSendSticker struct{}

type RequestMarkRead struct {
	MessageID string
	ChatJID   string
	SenderJID string
}

type RequestReact struct {
	MessageID string
	ChatJID   string
	Emoji     string
}

type RequestEdit struct {
	MessageID string
	ChatJID   string
	Message   string
	Text      string
}

type RequestDelete struct {
	MessageID string
	ChatJID   string
	SenderJID string
}

type RequestForward struct {
	MessageID string `json:"message_id"`
	ToChatJID string `json:"to_chat_jid"`
}

type RequestDownloadMedia struct {
	ChatJID   string `json:"chat_jid"`
	SenderJID string `json:"sender_jid"`
}

type RequestSetProfilePhoto struct {
	PhotoBase64 string `json:"photo_base64"`
}

type RequestContactSync struct {
	Phones []string `json:"phones"`
}

type RequestReply struct {
	ChatJID       string
	Message       string
	MessageID     string
	Text          string
	QuotedMessage *waE2E.Message
	TypingSimulation   *bool `json:"typing_simulation"`
	PresenceSimulation *bool `json:"presence_simulation"`
}

type RequestCreateGroup struct {
	Name         string
	Participants []string
	Description  string
	Photo        string
}

type RequestUpdateGroupSettings struct {
	Announce      *bool
	Locked        *bool
	MemberAddMode string
	JoinApproval  *bool
}

type RequestJoinGroupInvite struct {
	InviteCode string
	Inviter    string
	Expiration int64
}

type RequestSetGroupTopic struct {
	PreviousID string
	NewID      string
	Topic      string
}

type RequestPresence struct {
	Phone string
	State string
	Media string
}

type ResponseLogin struct {
	QRCode  string
	Timeout int
}

type ResponseLoginCode struct {
	PairCode string
	Timeout  int
}

type ResponseCheckPhone struct {
	IsRegistered bool
	JID          string
}

type ResponseUserInfo struct {
	VerifiedName string
	Status       string
	PictureID    string
	Devices      []string
}

type ResponseUserPicture struct {
	URL       string
	ID        string
	Type      string
	DirectURL string
}

// ============================================================
// NEW API REQUEST/RESPONSE TYPES
// ============================================================

// Call APIs
type RequestRejectCall struct {
	CallFrom string `json:"call_from"`
	CallID   string `json:"call_id"`
}

// Business APIs
type RequestBusinessProfile struct {
	JID string `json:"jid"`
}

type RequestResolveBusinessLink struct {
	Code string `json:"code"`
}

type ResponseBusinessProfile struct {
	JID         string                 `json:"jid"`
	Description string                 `json:"description"`
	Address     string                 `json:"address"`
	Email       string                 `json:"email"`
	Websites    []string               `json:"websites"`
	Categories  []map[string]string    `json:"categories"`
}

// Contact QR APIs
type RequestContactQRLink struct {
	Revoke bool `json:"revoke"`
}

type ResponseContactQRLink struct {
	Link string `json:"link"`
}

type RequestResolveContactQR struct {
	Code string `json:"code"`
}

type ResponseContactQRTarget struct {
	JID      string `json:"jid"`
	Type     string `json:"type"`
	PushName string `json:"push_name"`
}

// Bot APIs
type ResponseBotInfo struct {
	JID        string `json:"jid"`
	PluginType string `json:"plugin_type"`
	PluginName string `json:"plugin_name"`
}

type ResponseBotProfile struct {
	JID         string `json:"jid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PersonaID   string `json:"persona_id"`
}

// Presence APIs
type RequestSubscribePresence struct {
	JID string `json:"jid"`
}

// Newsletter APIs
type RequestNewsletterUpdates struct {
	Count int `json:"count"`
	Since int `json:"since"`
}

type RequestAcceptTOS struct {
	NoticeID string `json:"notice_id"`
	Stage    string `json:"stage"`
}

// History Sync APIs
type RequestHistorySyncRequest struct {
	ChatJID   string `json:"chat_jid"`
	SenderJID string `json:"sender_jid"`
	LastMsgID string `json:"last_msg_id"`
	Count     int    `json:"count"`
}

type RequestUnavailableMessage struct {
	ChatJID   string `json:"chat_jid"`
	SenderJID string `json:"sender_jid"`
	MessageID string `json:"message_id"`
}

// Media Retry APIs
type RequestMediaRetry struct {
	ChatJID   string `json:"chat_jid"`
	SenderJID string `json:"sender_jid"`
	MessageID string `json:"message_id"`
	MediaKey  string `json:"media_key"` // base64 encoded
}

// Utility APIs
type RequestSetPassive struct {
	Passive bool `json:"passive"`
}

type RequestWaitConnection struct {
	TimeoutSeconds int `json:"timeout_seconds"`
}

type ResponseWaitConnection struct {
	Connected bool `json:"connected"`
}

// Community APIs
type RequestUnlinkGroup struct {
	ParentJID string `json:"parent_jid"`
	ChildJID  string `json:"child_jid"`
}

// LID Mapping APIs
type RequestStoreLIDMapping struct {
	FirstJID  string `json:"first_jid"`
	SecondJID string `json:"second_jid"`
}

// ============================================================================
// History Sync APIs
// ============================================================================
type RequestBuildHistorySync struct {
	Count int `json:"count"` // Number of messages to sync
}

type ResponseHistorySync struct {
	Requested bool   `json:"requested"`
	Count     int    `json:"count"`
	Message   string `json:"message"`
}

// ============================================================================
// Per-Device Proxy APIs
// ============================================================================
type RequestSetProxy struct {
	ProxyURL string `json:"proxy_url"` // Empty string to disable
}

type ResponseGetProxy struct {
	ProxyURL string `json:"proxy_url"`
	Active   bool   `json:"active"`
}

// ============================================================================
// Poll Vote Decryption APIs
// ============================================================================
type RequestDecryptPollVote struct {
	MessageID string `json:"message_id"`
	ChatJID   string `json:"chat_jid"`
}

type ResponseDecryptPollVote struct {
	SelectedOptions []string `json:"selected_options"`
}

// ============================================================================
// Comment/Status Reply Encryption APIs
// ============================================================================
type RequestEncryptComment struct {
	MessageID string `json:"message_id"`
	Comment   string `json:"comment"`
}

type ResponseEncryptComment struct {
	EncryptedPayload string `json:"encrypted_payload"` // base64
}

type RequestDecryptComment struct {
	MessageID        string `json:"message_id"`
	EncryptedPayload string `json:"encrypted_payload"` // base64
}

type ResponseDecryptComment struct {
	Comment string `json:"comment"`
}

// ============================================================================
// Media Retry APIs
// ============================================================================
type RequestSendMediaRetryReceipt struct {
	ChatJID   string `json:"chat_jid"`
	SenderJID string `json:"sender_jid"`
	MessageID string `json:"message_id"`
	MediaKey  string `json:"media_key"` // base64
}

// ============================================================================
// Push Notification APIs
// ============================================================================
type RequestRegisterPushNotification struct {
	Platform string `json:"platform"` // "fcm" | "apns" | "webhook"
	Token    string `json:"token,omitempty"`
}

type ResponsePushNotificationStatus struct {
	Registered bool   `json:"registered"`
	Platform   string `json:"platform"`
}
