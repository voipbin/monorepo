package request

import "github.com/gofrs/uuid"

// V1DataDirectsPost is
// v1 data type request struct for
// /v1/directs POST
type V1DataDirectsPost struct {
	CustomerID   uuid.UUID `json:"customer_id"`
	ResourceType string    `json:"resource_type"`
	ResourceID   uuid.UUID `json:"resource_id"`
}
