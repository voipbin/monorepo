package request

import "github.com/gofrs/uuid"

// V1DataAIAuditsPost is the request body for POST /v1/aiaudits
type V1DataAIAuditsPost struct {
	CustomerID uuid.UUID `json:"customer_id,omitempty"`
	AIcallID   uuid.UUID `json:"aicall_id,omitempty"`
	Language   string    `json:"language,omitempty"`
}
