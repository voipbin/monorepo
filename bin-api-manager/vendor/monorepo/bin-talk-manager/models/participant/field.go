package participant

import "github.com/gofrs/uuid"

// Field represents database field names for participant
type Field string

const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"
	FieldChatID     Field = "chat_id"
	FieldOwnerType  Field = "owner_type"
	FieldOwnerID    Field = "owner_id"
	FieldTMJoined   Field = "tm_joined"
)

// FieldStruct defines filterable fields with their types
type FieldStruct struct {
	ID         uuid.UUID `filter:"id"`
	CustomerID uuid.UUID `filter:"customer_id"`
	ChatID     uuid.UUID `filter:"chat_id"`
	OwnerType  string    `filter:"owner_type"`
	OwnerID    uuid.UUID `filter:"owner_id"`
}

// GetDBFields returns list of database fields for SELECT queries
func GetDBFields() []string {
	return []string{
		"id",
		"customer_id",
		"chat_id",
		"owner_type",
		"owner_id",
		"tm_joined",
	}
}
