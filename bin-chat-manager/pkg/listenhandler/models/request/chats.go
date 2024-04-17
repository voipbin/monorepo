package request

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/chat"
)

// V1DataChatsPost is
// v1 data type request struct for
// /v1/chats POST
type V1DataChatsPost struct {
	CustomerID uuid.UUID `json:"customer_id"`
	Type       chat.Type `json:"type"`

	OwnerID        uuid.UUID   `json:"owner_id"`
	ParticipantIDs []uuid.UUID `json:"participant_ids"`

	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// V1DataChatsIDPut is
// v1 data type request struct for
// /v1/chats/{id} PUT
type V1DataChatsIDPut struct {
	Name   string `json:"name"`   // name
	Detail string `json:"detail"` // detail
}

// V1DataChatsIDParticipantIDsPost is
// v1 data type request struct for
// /v1/chats/<chat-id>/participant_ids POST
type V1DataChatsIDParticipantIDsPost struct {
	ParticipantID uuid.UUID `json:"participant_id"`
}

// V1DataChatsIDOwnerIDPut is
// v1 data type request struct for
// /v1/chats/{id}/owner_id PUT
type V1DataChatsIDOwnerIDPut struct {
	OwnerID uuid.UUID `json:"owner_id"`
}
