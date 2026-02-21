package message

import "github.com/gofrs/uuid"

// Field represents database field names for message
type Field string

const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"
	FieldChatID     Field = "chat_id"
	FieldParentID   Field = "parent_id"
	FieldOwnerType  Field = "owner_type"
	FieldOwnerID    Field = "owner_id"
	FieldType       Field = "type"
	FieldText       Field = "text"
	FieldMedias     Field = "medias"
	FieldMetadata   Field = "metadata"
	FieldTMCreate   Field = "tm_create"
	FieldTMUpdate   Field = "tm_update"
	FieldTMDelete   Field = "tm_delete"
	FieldDeleted    Field = "deleted" // Filter-only field
)

// FieldStruct defines filterable fields with their types
type FieldStruct struct {
	ID         uuid.UUID  `filter:"id"`
	CustomerID uuid.UUID  `filter:"customer_id"`
	ChatID     uuid.UUID  `filter:"chat_id"`
	ParentID   *uuid.UUID `filter:"parent_id"`
	OwnerType  string     `filter:"owner_type"`
	OwnerID    uuid.UUID  `filter:"owner_id"`
	Deleted    bool       `filter:"deleted"`
}

// GetDBFields returns list of database fields for SELECT queries
func GetDBFields() []string {
	return []string{
		"id",
		"customer_id",
		"chat_id",
		"parent_id",
		"owner_type",
		"owner_id",
		"type",
		"text",
		"medias",
		"metadata",
		"tm_create",
		"tm_update",
		"tm_delete",
	}
}
