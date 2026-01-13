package route

// Field represents database field names for Route
type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldName   Field = "name"   // name
	FieldDetail Field = "detail" // detail

	FieldProviderID Field = "provider_id" // provider_id
	FieldPriority   Field = "priority"    // priority

	FieldTarget Field = "target" // target

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
