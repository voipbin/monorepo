package request

import "github.com/gofrs/uuid"

// V1DataStreamingsPost is
// v1 data type request struct for
// /v1/streamings POST
type V1DataStreamingsPost struct {
	ReferenceID   uuid.UUID `json:"reference_id"`   // call/conference id
	Type          string    `json:"type"`           // reference type. call/conference
	Language      string    `json:"language"`       // BCP47 type's language code. en-US
	WebhookURI    string    `json:"webhook_uri"`    // webhook uri
	WebhookMethod string    `json:"webhook_method"` // webhook method
}
