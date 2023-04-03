package request

import (
	"github.com/gofrs/uuid"
	chatmedia "gitlab.com/voipbin/bin-manager/chat-manager.git/models/media"
	chatmessagechat "gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechat"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
)

// BodyChatmessagesPOST is rquest body define for
// POST /v1.0/chatmessages
type BodyChatmessagesPOST struct {
	ChatID uuid.UUID             `json:"chat_id"`
	Source commonaddress.Address `json:"source"`
	Type   chatmessagechat.Type  `json:"type"`
	Text   string                `json:"text"`
	Medias []chatmedia.Media     `json:"medias"`
}

// ParamChatmessagesGET is rquest param define for
// GET /v1.0/chatmessages
type ParamChatmessagesGET struct {
	ChatID string `form:"chat_id"`
	Pagination
}
