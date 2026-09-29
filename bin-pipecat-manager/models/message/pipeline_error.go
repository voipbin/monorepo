package message

import (
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
)

// ErrorCategory classifies a pipeline error into a customer-actionable class.
// The classification is deterministic (see pipecatcallhandler.classifyPipelineError).
type ErrorCategory string

// list of error categories.
//
// Only ErrorCategoryAuthentication, ErrorCategoryRateLimited, ErrorCategoryTimeout and
// ErrorCategoryUnknown ever reach a PipelineErrorEvent payload. ErrorCategoryFunctionCall and
// ErrorCategoryInternal are platform-side failures (our own tool handler raised, or a malformed
// RTVI envelope) and are only logged and counted.
const (
	ErrorCategoryAuthentication ErrorCategory = "authentication"
	ErrorCategoryRateLimited    ErrorCategory = "rate_limited"
	ErrorCategoryTimeout        ErrorCategory = "timeout"
	ErrorCategoryUnknown        ErrorCategory = "unknown"

	ErrorCategoryFunctionCall ErrorCategory = "function_call"
	ErrorCategoryInternal     ErrorCategory = "internal"
)

// PipelineErrorEvent is the event payload published when the pipecat runner reports an
// error (RTVI "error" frame) that should be surfaced to the customer on the owning aicall.
//
// The raw provider error text is deliberately NOT part of the payload: when the AI has no
// engine key the platform key is used, and the raw text can carry platform-internal detail.
// The raw text is logged by pipecat-manager instead. VOIP-1542.
type PipelineErrorEvent struct {
	CustomerID               uuid.UUID                 `json:"customer_id,omitempty"`
	PipecatcallID            uuid.UUID                 `json:"pipecatcall_id,omitempty"`
	PipecatcallReferenceType pipecatcall.ReferenceType `json:"pipecatcall_reference_type,omitempty"`
	PipecatcallReferenceID   uuid.UUID                 `json:"pipecatcall_reference_id,omitempty"`
	ActiveflowID             uuid.UUID                 `json:"activeflow_id,omitempty"`

	Category ErrorCategory `json:"category,omitempty"`
	Fatal    bool          `json:"fatal"`
}

// EventSubscriptionID returns the subscription address of the global topic exchange
// `bin-manager.event`. Like MemberSwitchedEvent it has no top-level `id`, so the address is the
// PipecatcallID: `pipecat-manager.pipeline.<pipecatcall-id>.error` lands in the same address
// space as the session's message, team and pipecatcall events.
//
// The receiver is a pointer because the event data reaches notifyhandler as a pointer and the
// eventtopic.SubscriptionIdentifier assertion matches the dynamic type.
func (h *PipelineErrorEvent) EventSubscriptionID() string {
	return h.PipecatcallID.String()
}
