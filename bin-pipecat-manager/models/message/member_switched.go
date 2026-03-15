package message

import (
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
)

// MemberSwitchedEvent is the event payload published when
// a team member transition occurs during an AI call.
type MemberSwitchedEvent struct {
	CustomerID               uuid.UUID                 `json:"customer_id,omitempty"`
	PipecatcallID            uuid.UUID                 `json:"pipecatcall_id,omitempty"`
	PipecatcallReferenceType pipecatcall.ReferenceType `json:"pipecatcall_reference_type,omitempty"`
	PipecatcallReferenceID   uuid.UUID                 `json:"pipecatcall_reference_id,omitempty"`
	ActiveflowID             uuid.UUID                 `json:"activeflow_id,omitempty"`
	TransitionFunctionName   string                    `json:"transition_function_name,omitempty"`
	FromMember               MemberInfo                `json:"from_member"`
	ToMember                 MemberInfo                `json:"to_member"`
}

// MemberInfo holds non-sensitive details about a team member.
type MemberInfo struct {
	ID          uuid.UUID `json:"id,omitempty"`
	Name        string    `json:"name,omitempty"`
	EngineModel string    `json:"engine_model,omitempty"`
	TTSType     string    `json:"tts_type,omitempty"`
	TTSVoiceID  string    `json:"tts_voice_id,omitempty"`
	STTType     string    `json:"stt_type,omitempty"`
}
