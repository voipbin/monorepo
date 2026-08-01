package execution

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Execution queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ScheduleID uuid.UUID `filter:"schedule_id"`
	Status     Status    `filter:"status"`
	Deleted    bool      `filter:"deleted"`
}
