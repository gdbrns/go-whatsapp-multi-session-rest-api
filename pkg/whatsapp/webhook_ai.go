package whatsapp

import (
	"encoding/json"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/encoding/protojson"
)

func buildAIRichResponseWebhookPayload(currentJID string, e *events.Message, richResponse *waE2E.AIRichResponseMessage) map[string]interface{} {
	payload := map[string]interface{}{
		"jid":                  currentJID,
		"message_id":           e.Info.ID,
		"from":                 e.Info.Sender.String(),
		"chat":                 e.Info.Chat.String(),
		"timestamp":            e.Info.Timestamp.Unix(),
		"is_from_me":           e.Info.IsFromMe,
		"message_type":         richResponse.GetMessageType().String(),
		"message_type_value":   int32(richResponse.GetMessageType()),
		"submessage_count":     len(richResponse.GetSubmessages()),
		"has_unified_response": richResponse.GetUnifiedResponse() != nil,
		"has_context_info":     richResponse.GetContextInfo() != nil,
	}

	options := protojson.MarshalOptions{UseProtoNames: true}
	if raw, err := options.Marshal(richResponse); err == nil {
		payload["rich_response"] = json.RawMessage(raw)
	}

	return payload
}
