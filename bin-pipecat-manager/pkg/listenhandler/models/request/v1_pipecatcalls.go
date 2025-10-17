package request

import (
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
)

// V1DataPipecatcallsPost is
// v1 data type request struct for
// /v1/pipecatcalls POST
type V1DataPipecatcallsPost struct {
	CustomerID    uuid.UUID                 `json:"customer_id,omitempty"`
	ActiveflowID  uuid.UUID                 `json:"activeflow_id,omitempty"`
	ReferenceType pipecatcall.ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID                 `json:"reference_id,omitempty"`
	LLM           pipecatcall.LLM           `json:"llm,omitempty"`
	STT           pipecatcall.STT           `json:"stt,omitempty"`
	TTS           pipecatcall.TTS           `json:"tts,omitempty"`
	VoiceID       string                    `json:"voice_id,omitempty"`
	Messages      []map[string]any          `json:"messages,omitempty"`
}
