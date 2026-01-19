package request

// V1DataMessagesIDReactionsPost represents the request body for POST /v1/messages/{id}/reactions
// Also used for DELETE /v1/messages/{id}/reactions
type V1DataMessagesIDReactionsPost struct {
	OwnerType string `json:"owner_type"`
	OwnerID   string `json:"owner_id"`
	Reaction  string `json:"reaction"`
}
