package rag

// Field defines the type for rag field names
type Field string

// List of rag fields
const (
	FieldID          Field = "id"
	FieldCustomerID  Field = "customer_id"
	FieldName        Field = "name"
	FieldDescription Field = "description"
	FieldTMCreate    Field = "tm_create"
	FieldTMUpdate    Field = "tm_update"
	FieldTMDelete    Field = "tm_delete"
)
