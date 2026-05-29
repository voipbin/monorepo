package aipromptproposal

import (
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Status represents the lifecycle state of a proposal record.
type Status string

const (
	StatusProgressing Status = "progressing" // Gemini call in flight
	StatusCompleted   Status = "completed"   // proposed_prompt ready; awaiting accept/reject
	StatusFailed      Status = "failed"      // generation error (terminal)
	StatusAccepted    Status = "accepted"    // merged into AI.InitPrompt (terminal)
	StatusRejected    Status = "rejected"    // user explicitly rejected (terminal)
	StatusExpired     Status = "expired"     // basis prompt drifted before accept (terminal)
)

// Error is a canonicalized string used in the error column.
type Error string

const (
	ErrorInvalidAuditSet            Error = "invalid_audit_set"
	ErrorAuditPromptVersionMismatch Error = "audit_prompt_version_mismatch"
	ErrorPromptVersionDrifted       Error = "prompt_version_drifted"
	ErrorEvaluatorUnavailable       Error = "evaluator_unavailable"
	ErrorInvalidEvaluatorResponse   Error = "invalid_evaluator_response"
	ErrorCancelled                  Error = "cancelled"
)

// AIPromptProposal represents one prompt-improvement proposal for one AI.
type AIPromptProposal struct {
	commonidentity.Identity // ID + CustomerID

	AIID                   uuid.UUID   `json:"ai_id"                              db:"ai_id,uuid"`
	AuditIDs               []uuid.UUID `json:"audit_ids,omitempty"                db:"audit_ids,json"`
	BasisPromptHistoryID   uuid.UUID   `json:"basis_prompt_history_id"            db:"basis_prompt_history_id,uuid"`
	OriginalPrompt         string      `json:"original_prompt,omitempty"          db:"original_prompt"`
	ProposedPrompt         string      `json:"proposed_prompt,omitempty"          db:"proposed_prompt"`
	Rationale              string      `json:"rationale,omitempty"                db:"rationale"`
	Status                 Status      `json:"status,omitempty"                   db:"status"`
	Error                  string      `json:"error,omitempty"                    db:"error"`
	AppliedPromptHistoryID uuid.UUID   `json:"applied_prompt_history_id,omitempty" db:"applied_prompt_history_id,uuid"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}
