package execution

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// TriggerType defines what triggered the execution.
type TriggerType string

// list of trigger types
const (
	TriggerTypeCron   TriggerType = "cron"
	TriggerTypeManual TriggerType = "manual"
)

// Status defines the execution's status.
type Status string

// list of execution statuses
const (
	StatusRunning   Status = "running"
	StatusSuccess   Status = "success"
	StatusFailed    Status = "failed"
	StatusAbandoned Status = "abandoned"
)

// Execution data model
type Execution struct {
	commonidentity.Identity

	ScheduleID uuid.UUID `json:"schedule_id" db:"schedule_id,uuid"` // schedule's id

	TriggerType TriggerType `json:"trigger_type" db:"trigger_type"` // what triggered the execution
	Status      Status      `json:"status" db:"status"`
	StatusCode  int         `json:"status_code" db:"status_code"` // RPC response status code(0 if transport error)
	Error       string      `json:"error" db:"error"`             // error string on failure/abandonment
	Result      string      `json:"result" db:"result"`           // RPC response body on success

	AttemptCount int `json:"attempt_count" db:"attempt_count"` // 1 + retries actually used
	DurationMS   int `json:"duration_ms" db:"duration_ms"`     // total wall time(ms) of the run

	TMScheduled *time.Time `json:"tm_scheduled" db:"tm_scheduled"` // the tm_next_run this row consumed
	TMDeadline  *time.Time `json:"tm_deadline" db:"tm_deadline"`   // reap deadline
	TMStart     *time.Time `json:"tm_start" db:"tm_start"`         // Started timestamp.
	TMEnd       *time.Time `json:"tm_end" db:"tm_end"`             // Ended timestamp.

	TMCreate *time.Time `json:"tm_create" db:"tm_create"` // Created timestamp.
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"` // Updated timestamp.
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"` // Deleted timestamp.
}
