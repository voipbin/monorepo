package execution

// Field represents a database field name for Execution model
type Field string

// List of Execution fields for database operations
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldScheduleID Field = "schedule_id"

	FieldTriggerType Field = "trigger_type"
	FieldStatus      Field = "status"
	FieldStatusCode  Field = "status_code"
	FieldError       Field = "error"
	FieldResult      Field = "result"

	FieldAttemptCount Field = "attempt_count"
	FieldDurationMS   Field = "duration_ms"

	FieldTMScheduled Field = "tm_scheduled"
	FieldTMDeadline  Field = "tm_deadline"
	FieldTMStart     Field = "tm_start"
	FieldTMEnd       Field = "tm_end"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	FieldDeleted Field = "deleted"
)
