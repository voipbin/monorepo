package device

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// WebhookMessage is the external representation of a Device. It has no token.
type WebhookMessage struct {
	commonidentity.Identity

	AgentID  uuid.UUID `json:"agent_id"` // owner agent id
	Platform Platform  `json:"platform"` // device platform

	TMCreate *time.Time `json:"tm_create"` // Created timestamp.
	TMUpdate *time.Time `json:"tm_update"` // Refreshed timestamp.
}

// ConvertWebhookMessage converts to the external representation.
func (h *Device) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		AgentID:  h.AgentID,
		Platform: h.Platform,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
	}
}
