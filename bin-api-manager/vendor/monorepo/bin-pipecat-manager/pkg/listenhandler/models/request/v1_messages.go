package request

import "github.com/gofrs/uuid"

// V1DataMessagesPost is
// v1 data type request struct for
// /v1/messages POST
type V1DataMessagesPost struct {
	PipecatcallID  uuid.UUID `json:"pipecatcall_id,omitempty"`
	MessageID      string    `json:"message_id,omitempty"`
	MessageText    string    `json:"message_text,omitempty"`
	RunImmediately bool      `json:"run_immediately,omitempty"`
	AudioResponse  bool      `json:"audio_response,omitempty"`
}
