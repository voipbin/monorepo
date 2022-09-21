package request

import (
	"github.com/gofrs/uuid"
	chatchat "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chat"
)

// BodyChatsPOST is rquest body define for POST /chats
type BodyChatsPOST struct {
	Type          chatchat.Type `json:"type"`
	OwnerID       uuid.UUID     `json:"owner_id"`
	ParticipantID []uuid.UUID   `json:"participant_ids"`
	Name          string        `json:"name"`
	Detail        string        `json:"detail"`
}

// ParamChatsGET is rquest param define for GET /chats
type ParamChatsGET struct {
	Pagination
}

// BodyChatsIDPUT is rquest body define for PUT /chats/{id}
type BodyChatsIDPUT struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// BodyChatsIDOwnerIDPUT is rquest body define for PUT /chats/{id}/owner_id
type BodyChatsIDOwnerIDPUT struct {
	OwnerID uuid.UUID `json:"owner_id"`
}

// BodyChatsIDParticipantIDsPOST is rquest body define for PUT /chats/{id}/participant_ids
type BodyChatsIDParticipantIDsPOST struct {
	ParticipantID uuid.UUID `json:"participant_id"`
}
