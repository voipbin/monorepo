package request

import "github.com/gofrs/uuid"

// V1DataRecordingsPost is
// v1 data type request struct for
// /v1/recordings POST
type V1DataRecordingsPost struct {
	CustomerID  uuid.UUID `json:"customer_id"`  // customer's id
	ReferenceID uuid.UUID `json:"reference_id"` // recording's id
	Language    string    `json:"language"`     // BCP47 type's language code. en-US
}

// V1DataCallRecordingsPost is
// v1 data type request struct for
// /v1/call-recordings POST
type V1DataCallRecordingsPost struct {
	CustomerID  uuid.UUID `json:"customer_id"`  // customer's id
	ReferenceID uuid.UUID `json:"reference_id"` // call's id
	Language    string    `json:"language"`     // BCP47 type's language code. en-US
}

// V1DataConferenceRecordingsPost is
// v1 data type request struct for
// /v1/conference-recordings POST
type V1DataConferenceRecordingsPost struct {
	CustomerID  uuid.UUID `json:"customer_id"`  // customer's id
	ReferenceID uuid.UUID `json:"reference_id"` // conference's id
	Language    string    `json:"language"`     // BCP47 type's language code. en-US
}
