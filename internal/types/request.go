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
	Latitude  float64
	Longitude float64
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
