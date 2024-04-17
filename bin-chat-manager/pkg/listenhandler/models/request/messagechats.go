package request

import (
	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"

	"monorepo/bin-chat-manager/models/media"
	"monorepo/bin-chat-manager/models/messagechat"
)

// V1DataMessagechatsPost is
// v1 data type request struct for
// /v1/messagechats POST
type V1DataMessagechatsPost struct {
	CustomerID  uuid.UUID             `json:"customer_id"`
	ChatID      uuid.UUID             `json:"chat_id"`
	Source      commonaddress.Address `json:"source"`
	MessageType messagechat.Type      `json:"message_type"`
	Text        string                `json:"text"`
	Medias      []media.Media         `json:"medias"`
}
