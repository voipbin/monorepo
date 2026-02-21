package request

import (
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
)

// V1DataSpeakingsPost is the request for POST /v1/speakings
type V1DataSpeakingsPost struct {
	CustomerID    uuid.UUID               `json:"customer_id,omitempty"`
	ReferenceType streaming.ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID               `json:"reference_id,omitempty"`
	Language      string                  `json:"language,omitempty"`
	Provider      string                  `json:"provider,omitempty"`
	VoiceID       string                  `json:"voice_id,omitempty"`
	Direction     streaming.Direction     `json:"direction,omitempty"`
}

// V1DataSpeakingsIDSayPost is the request for POST /v1/speakings/{id}/say
type V1DataSpeakingsIDSayPost struct {
	Text string `json:"text,omitempty"`
}
