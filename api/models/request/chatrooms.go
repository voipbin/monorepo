package request

import "github.com/gofrs/uuid"

// BodyChatroomsPOST is rquest body define for
// POST /v1.0/chatrooms
type BodyChatroomsPOST struct {
	ParticipantID []uuid.UUID `json:"participant_ids"`
	Name          string      `json:"name"`
	Detail        string      `json:"detail"`
}

// ParamChatroomsGET is rquest param define for
// GET /v1.0/chatrooms
type ParamChatroomsGET struct {
	OwnerID string `form:"owner_id"`
	Pagination
}

// BodyChatroomsIDPUT is rquest body define for
// PUT /v1.0/chatrooms/<chatroom-id>
type BodyChatroomsIDPUT struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}
