package request

import "github.com/gofrs/uuid"

// V1DataRecordingsPost is
// v1 data type request struct for
// /v1/recordings POST
type V1DataRecordingsPost struct {
	ReferenceID uuid.UUID `json:"reference_id"` // recording's id
	Language    string    `json:"language"`     // BCP47 type's language code. en-US
}

// V1DataCallRecordingsPost is
// v1 data type request struct for
// /v1/call-recordings POST
type V1DataCallRecordingsPost struct {
	ReferenceID   uuid.UUID `json:"reference_id"`   // call's id
	Language      string    `json:"language"`       // BCP47 type's language code. en-US
	WebhookURI    string    `json:"webhook_uri"`    // webhook destination uri
	WebhookMethod string    `json:"webhook_method"` // webhook method
}

// V1DataConferenceRecordingsPost is
// v1 data type request struct for
// /v1/conference-recordings POST
type V1DataConferenceRecordingsPost struct {
	ReferenceID   uuid.UUID `json:"reference_id"`   // conference's id
	Language      string    `json:"language"`       // BCP47 type's language code. en-US
	WebhookURI    string    `json:"webhook_uri"`    // webhook destination uri
	WebhookMethod string    `json:"webhook_method"` // webhook method
}
