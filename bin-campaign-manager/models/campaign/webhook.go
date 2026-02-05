package campaign

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	commonidentity.Identity

	Type Type `json:"type"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Status       Status    `json:"status"`
	ServiceLevel int       `json:"service_level"`
	EndHandle    EndHandle `json:"end_handle"`

	// action settings
	Actions []fmaction.Action `json:"actions"` // this actions will be stored to the flow

	// resource info
	OutplanID uuid.UUID `json:"outplan_id"`
	OutdialID uuid.UUID `json:"outdial_id"`
	QueueID   uuid.UUID `json:"queue_id"`

	NextCampaignID uuid.UUID `json:"next_campaign_id"`

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Campaign) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		Type: h.Type,

		Name:   h.Name,
		Detail: h.Detail,

		Status:       h.Status,
		ServiceLevel: h.ServiceLevel,
		EndHandle:    h.EndHandle,

		Actions: h.Actions,

		OutplanID: h.OutplanID,
		OutdialID: h.OutdialID,
		QueueID:   h.QueueID,

		NextCampaignID: h.NextCampaignID,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Campaign) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
