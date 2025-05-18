package account

import (
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Account defines
type Account struct {
	commonidentity.Identity

	Type Type `json:"type,omitempty"` // represent the type of account. could be message(sms/mms), line, etc.

	Name   string `json:"name,omitempty"`   // name of the account
	Detail string `json:"detail,omitempty"` // detail of the account

	Secret string `json:"secret,omitempty"` // secret
	Token  string `json:"token,omitempty"`  // usually api token

	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
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
	TypeLine Type = "line"
	TypeSMS  Type = "sms"
)
