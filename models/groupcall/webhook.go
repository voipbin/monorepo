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

	RingMethod   RingMethod   `json:"ring_method"`
	AnswerMethod AnswerMethod `json:"answer_method"`

	AnswerCallID uuid.UUID   `json:"answer_call_id"` // valid only when the answered_method  is hangup others
	CallIDs      []uuid.UUID `json:"call_ids"`

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

		RingMethod:   h.RingMethod,
		AnswerMethod: h.AnswerMethod,

		AnswerCallID: h.AnswerCallID,
		CallIDs:      h.CallIDs,

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
