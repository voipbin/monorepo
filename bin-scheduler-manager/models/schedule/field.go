package schedule

// Field represents a database field name for Schedule model
type Field string

// List of Schedule fields for database operations
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldName   Field = "name"
	FieldDetail Field = "detail"

	FieldType Field = "type"
	FieldCron Field = "cron"

	FieldTargetQueue    Field = "target_queue"
	FieldTargetURI      Field = "target_uri"
	FieldTargetMethod   Field = "target_method"
	FieldTargetDataType Field = "target_data_type"
	FieldTargetData     Field = "target_data"

	FieldTimeoutMS Field = "timeout_ms"
	FieldRetryMax  Field = "retry_max"
	FieldEnabled   Field = "enabled"

	FieldTMNextRun Field = "tm_next_run"
	FieldTMLastRun Field = "tm_last_run"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	FieldDeleted Field = "deleted"
)
