package groupcall

import (
	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
)

// Groupcall define
type Groupcall struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	Status Status    `json:"status"`
	FlowID uuid.UUID `json:"flow_id"`

	Source       *commonaddress.Address  `json:"source"`
	Destinations []commonaddress.Address `json:"destinations"`

	MasterCallID      uuid.UUID `json:"master_call_id,omitempty"`
	MasterGroupcallID uuid.UUID `json:"master_groupcall_id,omitempty"`

	RingMethod   RingMethod   `json:"ring_method"`
	AnswerMethod AnswerMethod `json:"answer_method"`

	AnswerCallID uuid.UUID   `json:"answer_call_id"` // represents answered call id.
	CallIDs      []uuid.UUID `json:"call_ids"`

	AnswerGroupcallID uuid.UUID   `json:"answer_groupcall_id"` // represents answered groupcall id
	GroupcallIDs      []uuid.UUID `json:"groupcall_ids"`

	CallCount      int `json:"call_count"`           // represent left number of calls for current dial
	GroupcallCount int `json:"groupcall_count"`      // represent left number of groupcalls for current dial
	DialIndex      int `json:"dial_index,omitempty"` // represent current dial index. valid only ringmethod is ringall

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
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
