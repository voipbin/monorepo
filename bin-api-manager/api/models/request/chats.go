package request

import (
	chatchat "monorepo/bin-chat-manager/models/chat"

	"github.com/gofrs/uuid"
)

// BodyChatsPOST is rquest body define for
// POST /v1.0/chats
type BodyChatsPOST struct {
	Type          chatchat.Type `json:"type"`
	OwnerID       uuid.UUID     `json:"owner_id"`
	ParticipantID []uuid.UUID   `json:"participant_ids"`
	Name          string        `json:"name"`
	Detail        string        `json:"detail"`
}

// ParamChatsGET is rquest param define for
// GET /v1.0/chats
type ParamChatsGET struct {
	Pagination
}

// BodyChatsIDPUT is rquest body define for
// PUT /v1.0/chats/<chat-id>
type BodyChatsIDPUT struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// BodyChatsIDOwnerIDPUT is rquest body define for
// PUT /v1.0/chats/<chat-id>/owner_id
type BodyChatsIDOwnerIDPUT struct {
	OwnerID uuid.UUID `json:"owner_id"`
}

// BodyChatsIDParticipantIDsPOST is rquest body define for
// PUT /v1.0/chats/<chat-id>/participant_ids
type BodyChatsIDParticipantIDsPOST struct {
	ParticipantID uuid.UUID `json:"participant_id"`
}
