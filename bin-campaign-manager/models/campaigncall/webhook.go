package campaigncall

import (
	"encoding/json"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	CampaignID uuid.UUID `json:"campaign_id"`

	OutplanID       uuid.UUID `json:"outplan_id"`
	OutdialID       uuid.UUID `json:"outdial_id"`
	OutdialTargetID uuid.UUID `json:"outdial_target_id"`
	QueueID         uuid.UUID `json:"queue_id"`

	ActiveflowID uuid.UUID `json:"activeflow_id"` // this is required
	FlowID       uuid.UUID `json:"flow_id"`

	ReferenceType ReferenceType `json:"reference_type"` // none or call
	ReferenceID   uuid.UUID     `json:"reference_id"`   // reference id

	Status Status `json:"status"`
	Result Result `json:"result"`

	Source           *commonaddress.Address `json:"source"`
	Destination      *commonaddress.Address `json:"destination"`
	DestinationIndex int                    `json:"destination_index"`
	TryCount         int                    `json:"try_count"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Campaigncall) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:         h.ID,
		CustomerID: h.CustomerID,

		CampaignID: h.CampaignID,

		OutplanID:       h.OutplanID,
		OutdialID:       h.OutdialID,
		OutdialTargetID: h.OutdialTargetID,
		QueueID:         h.QueueID,

		ActiveflowID: h.ActiveflowID,
		FlowID:       h.FlowID,

		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

		Status: h.Status,
		Result: h.Result,

		Source:           h.Source,
		Destination:      h.Destination,
		DestinationIndex: h.DestinationIndex,
		TryCount:         h.TryCount,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Campaigncall) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
