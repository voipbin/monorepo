package request

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"

	"github.com/gofrs/uuid"
)

// V1DataAccountsPost is
// v1 data type request struct for
// /v1/accounts POST
type V1DataConversationsPost struct {
	CustomerID uuid.UUID             `json:"customer_id,omitempty"`
	Name       string                `json:"name,omitempty"`
	Detail     string                `json:"detail,omitempty"`
	Type       conversation.Type     `json:"type,omitempty"`
	DialogID   string                `json:"dialog_id,omitempty"`
	Self       commonaddress.Address `json:"self,omitempty"`
	Peer       commonaddress.Address `json:"peer,omitempty"`
}

// V1DataConversationsIDMessagesPost is
// v1 data type request struct for
// /v1/conversations/<conversation-id>/messages POST
type V1DataConversationsIDMessagesPost struct {
	Text   string        `json:"text"`
	Medias []media.Media `json:"medias"`
}
