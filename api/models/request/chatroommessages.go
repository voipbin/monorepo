package request

import "github.com/gofrs/uuid"

// BodyChatroommessagesPOST is rquest body define for
// POST /v1.0/chatroommessages
type BodyChatroommessagesPOST struct {
	ChatroomID uuid.UUID `json:"chatroom_id"`
	Text       string    `json:"text"`
}

// ParamChatroommessagesGET is rquest param define for
// GET /v1.0/chatroommessages
type ParamChatroommessagesGET struct {
	ChatroomID string `form:"chatroom_id"`
	Pagination
}
