package session

// Field represents database field names for Session
type Field string

const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldWidgetID Field = "widget_id"
	FieldStatus   Field = "status"
	FieldPageURL  Field = "page_url"

	FieldActiveflowID Field = "activeflow_id"

	FieldTMLastActivity Field = "tm_last_activity"
	FieldTMCreate       Field = "tm_create"
	FieldTMUpdate       Field = "tm_update"
	FieldTMEnd          Field = "tm_end"
	FieldTMDelete       Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
