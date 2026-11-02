package request

// V1DataPinsPost represents the request body for POST /v1/pins
type V1DataPinsPost struct {
	CustomerID string `json:"customer_id"`
	OwnerType  string `json:"owner_type"`
	OwnerID    string `json:"owner_id"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
}

// V1DataPinsGet represents the request body for GET /v1/pins
// target_type and chat_id are optional filters.
type V1DataPinsGet struct {
	CustomerID string `json:"customer_id"`
	OwnerType  string `json:"owner_type"`
	OwnerID    string `json:"owner_id"`
	TargetType string `json:"target_type,omitempty"`
	ChatID     string `json:"chat_id,omitempty"`
}

// V1DataPinsIDDelete represents the request body for DELETE /v1/pins/{id}
type V1DataPinsIDDelete struct {
	CustomerID string `json:"customer_id"`
	OwnerType  string `json:"owner_type"`
	OwnerID    string `json:"owner_id"`
}
