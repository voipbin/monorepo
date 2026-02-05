package trunk

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-registrar-manager/models/sipauth"
)

// WebhookMessage defines
type WebhookMessage struct {
	commonidentity.Identity

	Name   string `json:"name"`
	Detail string `json:"detail"`

	DomainName string             `json:"domain_name"`
	AuthTypes  []sipauth.AuthType `json:"auth_types"`

	Username string `json:"username"`
	Password string `json:"password"`

	AllowedIPs []string `json:"allowed_ips"`

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Trunk) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

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
