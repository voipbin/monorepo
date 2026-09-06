package conversation

import "github.com/gofrs/uuid"

// Metadata is a generic, per-Conversation annotation payload, mirroring
// bin-customer-manager's Metadata pattern (models/customer/metadata.go):
// a single typed struct stored in one nullable JSON column, with its
// own dedicated update RPC rather than the general partial-update
// field allowlist.
type Metadata struct {
	// ContactCaseID claims this Conversation for a Case. Its only writer today
	// is bin-api-manager's Case message-reply path
	// (pkg/servicehandler/case_message.go), via ConversationV1ConversationUpdateMetadata;
	// flow-manager's case_create action does NOT set it, so consumers must not
	// rely on it being present for every conversation-origin Case (see
	// docs/plans/2026-09-05-insight-ai-conversation-listen-design.md §5.2).
	// Read-only from conversation-manager's own perspective: never read by
	// getExecuteMode or any flow/agent-routing dispatch decision.
	ContactCaseID *uuid.UUID `json:"contact_case_id,omitempty"`
}
