package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-tts-manager/models/tts"
)

// V1DataSpeechesPost is
// v1 data type request struct for
// /v1/speeches POST
type V1DataSpeechesPost struct {
	CallID   uuid.UUID    `json:"call_id"`
	Text     string       `json:"text"`
	Language string       `json:"language"`
	Provider tts.Provider `json:"provider,omitempty"`
	VoiceID  string       `json:"voice_id,omitempty"`
}
