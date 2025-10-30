package request

import (
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
)

// V1DataPipecatcallsPost is
// v1 data type request struct for
// /v1/pipecatcalls POST
type V1DataPipecatcallsPost struct {
	ID            uuid.UUID                 `json:"id,omitempty"`
	CustomerID    uuid.UUID                 `json:"customer_id,omitempty"`
	ActiveflowID  uuid.UUID                 `json:"activeflow_id,omitempty"`
	ReferenceType pipecatcall.ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID                 `json:"reference_id,omitempty"`
	LLMType       pipecatcall.LLMType       `json:"llm_type,omitempty"`
	LLMMessages   []map[string]any          `json:"llm_messages,omitempty"`
	STTType       pipecatcall.STTType       `json:"stt_type,omitempty"`
	TTSType       pipecatcall.TTSType       `json:"tts_type,omitempty"`
	TTSVoiceID    string                    `json:"tts_voice_id,omitempty"`
}
