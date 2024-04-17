package request

import "monorepo/bin-conversation-manager/models/media"

// V1DataConversationsIDMessagesPost is
// v1 data type request struct for
// /v1/conversations/<conversation-id>/messages POST
type V1DataConversationsIDMessagesPost struct {
	Text   string        `json:"text"`
	Medias []media.Media `json:"medias"`
}

// V1DataConversationsIDPut is
// v1 data type request struct for
// /v1/conversations/<conversation-id> PUT
type V1DataConversationsIDPut struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}
