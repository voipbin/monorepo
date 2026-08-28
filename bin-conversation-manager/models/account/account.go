package account

import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Account defines
type Account struct {
	commonidentity.Identity

	Type Type `json:"type,omitempty" db:"type"` // represent the type of account. could be message(sms/mms), line, etc.

	Name   string `json:"name,omitempty" db:"name"`     // name of the account
	Detail string `json:"detail,omitempty" db:"detail"` // detail of the account

	Secret string `json:"secret,omitempty" db:"secret"` // secret
	Token  string `json:"token,omitempty" db:"token"`   // usually api token

	ProviderData json.RawMessage `json:"provider_data,omitempty" db:"provider_data,json"`

	MessageFlowID uuid.UUID `json:"message_flow_id,omitempty" db:"message_flow_id,uuid"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// Field defines the fields for the Account entity.
type Field string

// List of account fields
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldType Field = "type"

	FieldName   Field = "name"
	FieldDetail Field = "detail"

	FieldSecret Field = "secret"
	FieldToken  Field = "token"

	FieldProviderData Field = "provider_data"

	FieldMessageFlowID Field = "message_flow_id"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)

// Type defines
type Type string

// list of types
const (
	TypeLine     Type = "line"
	TypeSMS      Type = "sms"
	TypeWhatsApp Type = "whatsapp"
)

// WhatsAppProviderData holds WhatsApp-specific credentials stored in provider_data JSON.
type WhatsAppProviderData struct {
	PhoneNumberID string `json:"phone_number_id"`
	AppSecret     string `json:"app_secret"`
}
