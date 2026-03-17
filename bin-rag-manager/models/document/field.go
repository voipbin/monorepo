package document

// Field defines the type for document field names
type Field string

const (
	FieldID            Field = "id"
	FieldCustomerID    Field = "customer_id"
	FieldRagID         Field = "rag_id"
	FieldName          Field = "name"
	FieldDocType       Field = "doc_type"
	FieldStorageFileID Field = "storage_file_id"
	FieldSourceURL     Field = "source_url"
	FieldStatus        Field = "status"
	FieldStatusMessage Field = "status_message"
	FieldTMCreate      Field = "tm_create"
	FieldTMUpdate      Field = "tm_update"
	FieldTMDelete      Field = "tm_delete"
)
