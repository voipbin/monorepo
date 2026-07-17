package widget

// Field represents database field names for Widget
type Field string

const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldName   Field = "name"
	FieldStatus Field = "status"

	FieldDirectID Field = "direct_id"
	FieldHash     Field = "direct_hash"

	FieldSessionFlowID Field = "session_flow_id"
	FieldMessageFlowID  Field = "message_flow_id"

	FieldSessionIdleTimeout Field = "session_idle_timeout"

	FieldThemeConfig Field = "theme_config"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
