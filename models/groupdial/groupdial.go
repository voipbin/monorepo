package groupdial

import (
	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
)

// GroupDial define
type GroupDial struct {
	ID           uuid.UUID              `json:"id"`
	CustomerID   uuid.UUID              `json:"customer_id"`
	Destination  *commonaddress.Address `json:"destination"`
	RingMethod   RingMethod             `json:"ring_method"`
	AnswerMethod AnswerMethod           `json:"answer_method"`
	CallIDs      []uuid.UUID            `json:"call_ids"`
}

// RingMethod define
type RingMethod string

// list of ring methods
const (
	RingMethodRingAll RingMethod = "ring all" // dial all
	RingMethodLinear  RingMethod = "linear"
)

// AnswerMethod define
type AnswerMethod string

// list of answer methods
const (
	AnswerMethodNone         AnswerMethod = ""              // do nothing
	AnswerMethodHangupOthers AnswerMethod = "hangup others" // hangup others
)
