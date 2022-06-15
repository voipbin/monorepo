package request

import "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"

// V1DataConversationIDMessagePost is
// v1 data type request struct for
// /v1/conversations/<conversation-id>/messages POST
type V1DataConversationIDMessagePost struct {
	Text   string        `json:"text"`
	Medias []media.Media `json:"medias"`
}
