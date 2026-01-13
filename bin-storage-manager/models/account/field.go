package account

// Field represents a typed field name for database operations
type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldTotalFileCount Field = "total_file_count" // total_file_count
	FieldTotalFileSize  Field = "total_file_size"  // total_file_size

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
