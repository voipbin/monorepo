package groupcall

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Groupcall define
type Groupcall struct {
	commonidentity.Identity
	commonidentity.Owner

	Status Status    `json:"status,omitempty" db:"status"`
	FlowID uuid.UUID `json:"flow_id,omitempty" db:"flow_id,uuid"`

	Source       *commonaddress.Address  `json:"source,omitempty" db:"source,json"`
	Destinations []commonaddress.Address `json:"destinations,omitempty" db:"destinations,json"`

	MasterCallID      uuid.UUID `json:"master_call_id,omitempty" db:"master_call_id,uuid"`
	MasterGroupcallID uuid.UUID `json:"master_groupcall_id,omitempty" db:"master_groupcall_id,uuid"`

	RingMethod   RingMethod   `json:"ring_method,omitempty" db:"ring_method"`
	AnswerMethod AnswerMethod `json:"answer_method,omitempty" db:"answer_method"`

	AnswerCallID uuid.UUID   `json:"answer_call_id,omitempty" db:"answer_call_id,uuid"` // represents answered call id.
	CallIDs      []uuid.UUID `json:"call_ids,omitempty" db:"call_ids,json"`

	AnswerGroupcallID uuid.UUID   `json:"answer_groupcall_id,omitempty" db:"answer_groupcall_id,uuid"` // represents answered groupcall id
	GroupcallIDs      []uuid.UUID `json:"groupcall_ids,omitempty" db:"groupcall_ids,json"`

	CallCount      int `json:"call_count,omitempty" db:"call_count"`           // represent left number of calls for current dial
	GroupcallCount int `json:"groupcall_count,omitempty" db:"groupcall_count"` // represent left number of groupcalls for current dial
	DialIndex      int `json:"dial_index,omitempty" db:"dial_index"`           // represent current dial index. valid only ringmethod is ringall

	// timestamp
	TMCreate string `json:"tm_create,omitempty" db:"tm_create"`
	TMUpdate string `json:"tm_update,omitempty" db:"tm_update"`
	TMDelete string `json:"tm_delete,omitempty" db:"tm_delete"`
}

// Status define
type Status string

// list of status
const (
	StatusProgressing Status = "progressing"
	StatusHangingup   Status = "hangingup"
	StatusHangup      Status = "hangup"
)

// RingMethod define
type RingMethod string

// list of ring methods
const (
	RingMethodNone    RingMethod = ""
	RingMethodRingAll RingMethod = "ring_all" // dial all
	RingMethodLinear  RingMethod = "linear"
)

// AnswerMethod define
type AnswerMethod string

// list of answer methods
const (
	AnswerMethodNone         AnswerMethod = ""              // do nothing
	AnswerMethodHangupOthers AnswerMethod = "hangup_others" // hangup others
)
