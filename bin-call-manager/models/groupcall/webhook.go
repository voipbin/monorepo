package groupcall

import (
	"encoding/json"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	commonidentity.Identity
	commonidentity.Owner

	Status Status    `json:"status,omitempty"`
	FlowID uuid.UUID `json:"flow_id,omitempty"`

	Source       *commonaddress.Address  `json:"source,omitempty"`
	Destinations []commonaddress.Address `json:"destinations,omitempty"`

	MasterCallID      uuid.UUID `json:"master_call_id,omitempty"`
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
	TMCreate *time.Time `json:"tm_create,omitempty"`
	TMUpdate *time.Time `json:"tm_update,omitempty"`
	TMDelete *time.Time `json:"tm_delete,omitempty"`
}

// ConvertWebhookMessage converts to the event
func (h *Groupcall) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,
		Owner:    h.Owner,

		Status: h.Status,
		FlowID: h.FlowID,

		Source:       h.Source,
		Destinations: h.Destinations,

		MasterCallID:      h.MasterCallID,
		MasterGroupcallID: h.MasterGroupcallID,

		RingMethod:   h.RingMethod,
		AnswerMethod: h.AnswerMethod,

		AnswerCallID: h.AnswerCallID,
		CallIDs:      h.CallIDs,

		AnswerGroupcallID: h.AnswerCallID,
		GroupcallIDs:      h.GroupcallIDs,

		CallCount:      h.CallCount,
		GroupcallCount: h.GroupcallCount,
		DialIndex:      h.DialIndex,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generate WebhookEvent
func (h *Groupcall) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
