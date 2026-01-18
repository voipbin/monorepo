package chat

import "github.com/gofrs/uuid"

// Field represents database field names for talk
type Field string

const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"
	FieldType       Field = "type"
	FieldName       Field = "name"
	FieldDetail     Field = "detail"
	FieldTMCreate   Field = "tm_create"
	FieldTMUpdate   Field = "tm_update"
	FieldTMDelete   Field = "tm_delete"
	FieldDeleted    Field = "deleted"    // Filter-only field
	FieldOwnerType  Field = "owner_type" // Filter-only field (from participants table)
	FieldOwnerID    Field = "owner_id"   // Filter-only field (from participants table)
)

// FieldStruct defines filterable fields with their types
type FieldStruct struct {
	ID         uuid.UUID `filter:"id"`
	CustomerID uuid.UUID `filter:"customer_id"`
	Type       string    `filter:"type"`
	Name       string    `filter:"name"`
	Detail     string    `filter:"detail"`
	Deleted    bool      `filter:"deleted"`
	OwnerType  string    `filter:"owner_type"`  // Filter by participant owner type
	OwnerID    uuid.UUID `filter:"owner_id"`    // Filter by participant owner ID
}

// GetDBFields returns list of database fields for SELECT queries
func GetDBFields() []string {
	return []string{
		"id",
		"customer_id",
		"type",
		"name",
		"detail",
		"tm_create",
		"tm_update",
		"tm_delete",
	}
}
