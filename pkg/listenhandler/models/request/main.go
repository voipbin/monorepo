package request

import "github.com/gofrs/uuid"

// V1DataSpeechesPost is
// v1 data type request struct for
// /v1/speeches POST
type V1DataSpeechesPost struct {
	CallID   uuid.UUID `json:"call_id"`
	Text     string    `json:"text"`
	Gender   string    `json:"gender"`
	Language string    `json:"language"`
}
