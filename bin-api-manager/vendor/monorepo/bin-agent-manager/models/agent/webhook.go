package agent

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

	Username string `json:"username"` // agent's username

	Name   string `json:"name"`   // agent's name
	Detail string `json:"detail"` // agent's detail

	RingMethod RingMethod `json:"ring_method"` // agent's ring method

	Status     Status                  `json:"status"`     // agent's status
	Permission Permission              `json:"permission"` // agent's permission.
	TagIDs     []uuid.UUID             `json:"tag_ids"`    // agent's tag ids
	Addresses  []commonaddress.Address `json:"addresses"`  // agent's endpoint addresses

	TMCreate *time.Time `json:"tm_create,omitempty"` // Created timestamp.
	TMUpdate *time.Time `json:"tm_update,omitempty"` // Updated timestamp.
	TMDelete *time.Time `json:"tm_delete,omitempty"` // Deleted timestamp.
}

// ConvertWebhookMessage converts to the event
func (h *Agent) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		Username: h.Username,

		Name:   h.Name,
		Detail: h.Detail,

		RingMethod: h.RingMethod,

		Status:     h.Status,
		Permission: h.Permission,
		TagIDs:     h.TagIDs,
		Addresses:  h.Addresses,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Agent) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
