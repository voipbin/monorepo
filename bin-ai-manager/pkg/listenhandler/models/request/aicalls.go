package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
)

// V1DataAIcallsPost is
// v1 data type request struct for
// /v1/aicalls POST
type V1DataAIcallsPost struct {
	AssistanceType aicall.AssistanceType `json:"assistance_type,omitempty"`
	AssistanceID   uuid.UUID             `json:"assistance_id,omitempty"`

	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`

	ReferenceType aicall.ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID            `json:"reference_id,omitempty"`

}

// V1DataAIcallsIDMessagesPost is
// v1 data type request struct for
// /v1/aicalls/<ai-id>/messages POST
type V1DataAIcallsIDMessagesPost struct {
	Role aicall.MessageRole `json:"role,omitempty"`
	Text string             `json:"text,omitempty"`
}

// V1DataAIcallsIDToolExecutePost is
// v1 data type request struct for
// /v1/aicalls/<ai-id>/tool_execute POST
type V1DataAIcallsIDToolExecutePost struct {
	ID       string               `json:"id,omitempty"`
	Type     message.ToolType     `json:"type,omitempty"`
	Function message.FunctionCall `json:"function,omitempty"`

	// PipecatcallID is the pipecatcall session this tool call arrived on.
	//
	// DELIBERATELY OPTIONAL (omitempty, no required-field validation): during a
	// rolling deploy an old bin-pipecat-manager sends no such field, which
	// unmarshals to uuid.Nil. Every consumer treats uuid.Nil as "this is the
	// agent's own Q&A turn" -- the fail-safe direction, since guessing that way
	// costs at most one rejected notify_agent call, whereas guessing "listen
	// turn" would permanently mistag real conversational content. The reverse
	// direction (new pipecat-manager, old ai-manager) is safe too: an old
	// ai-manager ignores an unknown JSON field. No deployment order is forced.
	//
	// See docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.4.3a.
	PipecatcallID uuid.UUID `json:"pipecatcall_id,omitempty"`
}
