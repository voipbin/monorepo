package aiaudit

import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Status represents the lifecycle state of an audit record.
type Status string

const (
	StatusProgressing Status = "progressing"
	StatusCompleted   Status = "completed"
	StatusFailed      Status = "failed"
)

// Error is a canonicalized string used in the error field.
type Error string

const (
	ErrorInvalidCallMetadata       Error = "invalid_call_metadata"
	ErrorPromptSnapshotNotFound    Error = "prompt_snapshot_not_found"
	ErrorPromptSnapshotNoHistoryID Error = "prompt_snapshot_has_no_history_id"
	ErrorInvalidEvaluatorResponse  Error = "invalid_evaluator_response"
	ErrorEvaluatorUnavailable      Error = "evaluator_unavailable"
	ErrorCancelled                 Error = "cancelled"
)

// AIAudit represents a single on-demand audit record for one AI in one call.
type AIAudit struct {
	commonidentity.Identity

	AIcallID        uuid.UUID       `json:"aicall_id,omitempty"         db:"aicall_id,uuid"`
	AIID            uuid.UUID       `json:"ai_id,omitempty"             db:"ai_id,uuid"`
	PromptHistoryID uuid.UUID       `json:"prompt_history_id,omitempty" db:"prompt_history_id,uuid"`

	Status       Status          `json:"status,omitempty"  db:"status"`
	OverallScore *int            `json:"overall_score"     db:"overall_score"`
	Evaluation   json.RawMessage `json:"evaluation"        db:"evaluation,json"`
	Language     string          `json:"language,omitempty" db:"language"`
	Error        string          `json:"error,omitempty"   db:"error"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}
