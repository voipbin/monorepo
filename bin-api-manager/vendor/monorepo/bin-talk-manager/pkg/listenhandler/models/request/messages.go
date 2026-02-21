package request

import "monorepo/bin-talk-manager/models/message"

// V1DataMessagesPost represents the request body for POST /v1/messages
type V1DataMessagesPost struct {
	ChatID    string          `json:"chat_id"`
	ParentID  *string         `json:"parent_id,omitempty"`
	OwnerType string          `json:"owner_type"`
	OwnerID   string          `json:"owner_id"`
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Medias    []message.Media `json:"medias"`
}
