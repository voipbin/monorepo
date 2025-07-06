package request

import (
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
)

// V1DataStreamingsPost is
// v1 data type request struct for
// /v1/streamings POST
type V1DataStreamingsPost struct {
	CustomerID    uuid.UUID               `json:"customer_id,omitempty"`
	ReferenceType streaming.ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID               `json:"reference_id,omitempty"`
	Language      string                  `json:"language,omitempty"`
	Gender        streaming.Gender        `json:"gender,omitempty"`
	Direction     streaming.Direction     `json:"direction,omitempty"` // Direction of the streaming
}

// V1DataStreamingsIDSayPost is
// v1 data type request struct for
// /v1/streamings/<id>/say POST
type V1DataStreamingsIDSayPost struct {
	Text string `json:"text,omitempty"`
}
