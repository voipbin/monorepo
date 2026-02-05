package conversation

import (
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Conversation represents a conversation entity with associated metadata and details.
// It includes information about the account, reference type, and reference ID,
// as well as addresses for the self and peer participants. Additionally, it tracks
// timestamps for creation, update, and deletion events.
type Conversation struct {
	commonidentity.Identity
	commonidentity.Owner

	AccountID uuid.UUID `json:"account_id,omitempty" db:"account_id,uuid"`

	Name   string `json:"name,omitempty" db:"name"`
	Detail string `json:"detail,omitempty" db:"detail"`

	Type Type `json:"type,omitempty" db:"type"` // represent the type of conversation. could be message(sms/mms), line, etc.

	DialogID string `json:"dialog_id,omitempty" db:"dialog_id"` // represent the id of referenced conversation transaction. for the Line could be chatroom id, for sms/mms, could be empty.

	Self commonaddress.Address `json:"self,omitempty" db:"self,json"` // self address
	Peer commonaddress.Address `json:"peer,omitempty" db:"peer,json"` // peer address

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

type Field string

const (
	FieldDeleted Field = "deleted"

	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldOwnerType Field = "owner_type"
	FieldOwnerID   Field = "owner_id"

	FieldAccountID Field = "account_id"

	FieldName   Field = "name"
	FieldDetail Field = "detail"

	FieldType Field = "type"

	FieldDialogID Field = "dialog_id"

	FieldSelf Field = "self"
	FieldPeer Field = "peer"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"
)

// Type defines
type Type string

// list of reference types
const (
	TypeNone    Type = ""
	TypeMessage Type = "message" // sms, mms
	TypeLine    Type = "line"
)
