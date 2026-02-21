package request

// V1DataChatsIDParticipantsPost represents the request body for POST /v1/chats/{id}/participants
type V1DataChatsIDParticipantsPost struct {
	OwnerType string `json:"owner_type"`
	OwnerID   string `json:"owner_id"`
}
