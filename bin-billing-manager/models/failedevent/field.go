package failedevent

// Field represents a failed event field for database queries.
type Field string

// List of fields
const (
	FieldID             Field = "id"
	FieldRetryCount     Field = "retry_count"
	FieldNextRetryAt    Field = "next_retry_at"
	FieldStatus         Field = "status"
	FieldTMUpdate       Field = "tm_update"
)
