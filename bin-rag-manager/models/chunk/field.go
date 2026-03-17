package chunk

// Field defines the type for chunk field names
type Field string

const (
	FieldID           Field = "id"
	FieldDocumentID   Field = "document_id"
	FieldRagID        Field = "rag_id"
	FieldCustomerID   Field = "customer_id"
	FieldChunkIndex   Field = "chunk_index"
	FieldText         Field = "text"
	FieldSectionTitle Field = "section_title"
	FieldEmbedding    Field = "embedding"
	FieldTokenCount   Field = "token_count"
	FieldTMCreate     Field = "tm_create"
	FieldTMDelete     Field = "tm_delete"
)
