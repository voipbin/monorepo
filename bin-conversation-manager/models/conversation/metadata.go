package conversation

import "github.com/gofrs/uuid"

// Metadata is a generic, per-Conversation annotation payload, mirroring
// bin-customer-manager's Metadata pattern (models/customer/metadata.go):
// a single typed struct stored in one nullable JSON column, with its
// own dedicated update RPC rather than the general partial-update
// field allowlist.
type Metadata struct {
	// ContactCaseID is set by bin-contact-manager, from either write path
	// described in docs/plans/2026-07-07-contact-case-management-design.md
	// §4.3, to claim this Conversation for a Case. Read-only from
	// conversation-manager's own perspective: never read by
	// getExecuteMode or any flow/agent-routing dispatch decision.
	ContactCaseID *uuid.UUID `json:"contact_case_id,omitempty"`
}
