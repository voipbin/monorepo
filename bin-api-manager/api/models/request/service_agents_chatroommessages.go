package request

import (
	chatmedia "monorepo/bin-chat-manager/models/media"

	"github.com/gofrs/uuid"
)

// ParamServiceAgentsChatroommessagesGET is rquest param define for
// GET /v1.0/service_agents/chatroommessages
type ParamServiceAgentsChatroommessagesGET struct {
	ChatroomID string `form:"chatroom_id"`
	Pagination
}

// BodyChatroommessagesPOST is rquest body define for
// POST /v1.0/service_agents/chatroommessages
type BodyServiceAgentsChatroommessagesPOST struct {
	ChatroomID uuid.UUID         `json:"chatroom_id"`
	Text       string            `json:"text"`
	Medias     []chatmedia.Media `json:"medias"`
}
