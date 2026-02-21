package request

// V1DataChatsParticipant represents a participant in a chat creation request
type V1DataChatsParticipant struct {
	OwnerType string `json:"owner_type"`
	OwnerID   string `json:"owner_id"`
}

// V1DataChatsPost represents the request body for POST /v1/chats
type V1DataChatsPost struct {
	CustomerID   string                   `json:"customer_id"`
	Type         string                   `json:"type"`
	Name         string                   `json:"name"`
	Detail       string                   `json:"detail"`
	CreatorType  string                   `json:"creator_type"`
	CreatorID    string                   `json:"creator_id"`
	Participants []V1DataChatsParticipant `json:"participants"`
}

// V1DataChatsIDPut represents the request body for PUT /v1/chats/{id}
type V1DataChatsIDPut struct {
	Name   *string `json:"name"`
	Detail *string `json:"detail"`
}
