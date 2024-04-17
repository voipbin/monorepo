package request

import (
	commonaddress "monorepo/bin-common-handler/models/address"

	chatmedia "monorepo/bin-chat-manager/models/media"
	chatmessagechat "monorepo/bin-chat-manager/models/messagechat"

	"github.com/gofrs/uuid"
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
