package request

import (
	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechat"
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
