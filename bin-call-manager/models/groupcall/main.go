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

	Status Status    `json:"status,omitempty"`
	FlowID uuid.UUID `json:"flow_id,omitempty"`

	Source       *commonaddress.Address  `json:"source,omitempty"`
	Destinations []commonaddress.Address `json:"destinations,omitempty"`

	MasterCallID      uuid.UUID `json:"master_call_id,omitempty,omitempty"`
	MasterGroupcallID uuid.UUID `json:"master_groupcall_id,omitempty"`

	RingMethod   RingMethod   `json:"ring_method,omitempty"`
	AnswerMethod AnswerMethod `json:"answer_method,omitempty"`

	AnswerCallID uuid.UUID   `json:"answer_call_id,omitempty"` // represents answered call id.
	CallIDs      []uuid.UUID `json:"call_ids,omitempty"`

	AnswerGroupcallID uuid.UUID   `json:"answer_groupcall_id,omitempty"` // represents answered groupcall id
	GroupcallIDs      []uuid.UUID `json:"groupcall_ids,omitempty"`

	CallCount      int `json:"call_count,omitempty"`      // represent left number of calls for current dial
	GroupcallCount int `json:"groupcall_count,omitempty"` // represent left number of groupcalls for current dial
	DialIndex      int `json:"dial_index,omitempty"`      // represent current dial index. valid only ringmethod is ringall

	// timestamp
	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
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
