package trunk

import (
	"encoding/json"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID         uuid.UUID `json:"id,omitempty"`
	CustomerID uuid.UUID `json:"customer_id,omitempty"`

	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	DomainName string     `json:"domain_name,omitempty"`
	AuthTypes  []AuthType `json:"auth_types,omitempty"`

	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	AllowedIPs []string `json:"allowed_ips,omitempty"`

	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
}

// ConvertWebhookMessage converts to the event
func (h *Trunk) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:         h.ID,
		CustomerID: h.CustomerID,

		Name:   h.Name,
		Detail: h.Detail,

		DomainName: h.DomainName,
		AuthTypes:  h.AuthTypes,

		Username: h.Username,
		Password: h.Password,

		AllowedIPs: h.AllowedIPs,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Trunk) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
