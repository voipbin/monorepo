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

// EventSubscriptionID returns the subscription address of the global topic exchange
// `bin-manager.event` (VOIP-1404 / VOIP-1405). MemberSwitchedEvent has no top-level `id` at all,
// so without this override the default JSON fallback would resolve to the `-` placeholder and the
// event would be unreachable by any instance binding. The address is the PipecatcallID, the same
// one every message event of the session carries, so `pipecat-manager.team.<pipecatcall-id>.#`
// lands in the same address space as the message and pipecatcall namespaces.
//
// The receiver is a pointer because the event data reaches notifyhandler as a pointer and the
// eventtopic.SubscriptionIdentifier assertion matches the dynamic type.
func (h *MemberSwitchedEvent) EventSubscriptionID() string {
	return h.PipecatcallID.String()
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
