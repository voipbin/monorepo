package outboundconfig

import (
	"time"

	"github.com/gofrs/uuid"
)

// WebhookMessage is the external-facing representation of OutboundConfig.
// Always return this type through the public API — never the internal struct.
// ConvertWebhookMessage exists to future-proof against internal-only fields.
type WebhookMessage struct {
	ID                   uuid.UUID  `json:"id"`
	CustomerID           uuid.UUID  `json:"customer_id"`
	Name                 string     `json:"name"`
	Detail               string     `json:"detail"`
	DestinationWhitelist []string   `json:"destination_whitelist"`
	Codecs               string     `json:"codecs"`
	TMCreate             *time.Time `json:"tm_create"`
	TMUpdate             *time.Time `json:"tm_update"`
	TMDelete             *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts an OutboundConfig to its external form.
func ConvertWebhookMessage(c *OutboundConfig) *WebhookMessage {
	if c == nil {
		return nil
	}
	return &WebhookMessage{
		ID:                   c.ID,
		CustomerID:           c.CustomerID,
		Name:                 c.Name,
		Detail:               c.Detail,
		DestinationWhitelist: c.DestinationWhitelist,
		Codecs:               c.Codecs,
		TMCreate:             c.TMCreate,
		TMUpdate:             c.TMUpdate,
		TMDelete:             c.TMDelete,
	}
}
