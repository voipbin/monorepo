package groupcall

import (
	"encoding/json"

	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
)

// WebhookMessage defines
type WebhookMessage struct {
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

// ConvertWebhookMessage converts to the event
func (h *Groupcall) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:         h.ID,
		CustomerID: h.CustomerID,

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
