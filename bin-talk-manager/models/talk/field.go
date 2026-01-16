package talk

import "github.com/gofrs/uuid"

// Field represents database field names for talk
type Field string

const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"
	FieldType       Field = "type"
	FieldTMCreate   Field = "tm_create"
	FieldTMUpdate   Field = "tm_update"
	FieldTMDelete   Field = "tm_delete"
	FieldDeleted    Field = "deleted" // Filter-only field
)

// FieldStruct defines filterable fields with their types
type FieldStruct struct {
	ID         uuid.UUID `filter:"id"`
	CustomerID uuid.UUID `filter:"customer_id"`
	Type       string    `filter:"type"`
	Deleted    bool      `filter:"deleted"`
}

// GetDBFields returns list of database fields for SELECT queries
func GetDBFields() []string {
	return []string{
		"id",
		"customer_id",
		"type",
		"tm_create",
		"tm_update",
		"tm_delete",
	}
}
