package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-transcribe-manager/models/transcribe"
)

// V1DataTranscribesPost is
// v1 data type request struct for
// /v1/transcribes POST
type V1DataTranscribesPost struct {
	CustomerID    uuid.UUID                `json:"customer_id,omitempty"` // customer id
	ActiveflowID  uuid.UUID                `json:"activeflow_id,omitempty"`
	OnEndFlowID   uuid.UUID                `json:"on_end_flow_id,omitempty"`
	ReferenceType transcribe.ReferenceType `json:"reference_type,omitempty"` // reference type. call/conference/recording, ...
	ReferenceID   uuid.UUID                `json:"reference_id,omitempty"`   // reference id
	Language      string                   `json:"language,omitempty"`       // BCP47 type's language code. en-US
	Direction     transcribe.Direction     `json:"direction,omitempty"`
}

// V1DataTranscribesIDHealthCheckPost is
// v1 data type request struct for
// /v1/transcribes/<transcribe-id>/health-check POST
type V1DataTranscribesIDHealthCheckPost struct {
	RetryCount int `json:"retry_count,omitempty"`
}
