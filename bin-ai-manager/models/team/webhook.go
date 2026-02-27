package team

import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

// WebhookMessage is the external-facing representation of a Team.
type WebhookMessage struct {
	commonidentity.Identity

	Name          string    `json:"name,omitempty"`
	Detail        string    `json:"detail,omitempty"`
	StartMemberID uuid.UUID `json:"start_member_id,omitempty"`
	Members       []Member  `json:"members,omitempty"`

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts the internal Team to an external WebhookMessage.
func (h *Team) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		Name:          h.Name,
		Detail:        h.Detail,
		StartMemberID: h.StartMemberID,
		Members:       h.Members,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generate WebhookEvent
func (h *Team) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
