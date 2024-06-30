package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-chat-manager/models/chat"
)

// V1DataChatsPost is
// v1 data type request struct for
// /v1/chats POST
type V1DataChatsPost struct {
	CustomerID uuid.UUID `json:"customer_id"`
	Type       chat.Type `json:"type"`

	RoomOwnerID    uuid.UUID   `json:"room_owner_id"`
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

// V1DataChatsIDRoomOwnerIDPut is
// v1 data type request struct for
// /v1/chats/{id}/room_owner_id PUT
type V1DataChatsIDRoomOwnerIDPut struct {
	RoomOwnerID uuid.UUID `json:"room_owner_id"`
}
