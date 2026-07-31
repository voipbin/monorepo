package schedule

// list of event types
const (
	EventTypeScheduleCreated string = "schedule_created"
	EventTypeScheduleUpdated string = "schedule_updated"
	EventTypeScheduleDeleted string = "schedule_deleted"

	EventTypeExecutionSucceeded string = "execution_succeeded"
	EventTypeExecutionFailed    string = "execution_failed"
)
