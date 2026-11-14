package provider

import (
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-number-manager/models/number"
)

// WebhookMessage is the externally exposed view of a provider setting. It never carries secret
// material: only whether the secret variable of the provider holds a value (has_secret).
type WebhookMessage struct {
	ProviderName number.ProviderName `json:"provider_name"`

	Enabled  bool              `json:"enabled"`
	Priority int               `json:"priority"`
	Config   map[string]string `json:"config,omitempty"`

	HasSecret        bool             `json:"has_secret"`
	CredentialSource CredentialSource `json:"credential_source"`
	Usable           bool             `json:"usable"`

	UpdatedBy *uuid.UUID `json:"updated_by,omitempty"`

	TMUpdate *time.Time `json:"tm_update,omitempty"`
}

// ConvertWebhookMessage converts the admin view to the externally exposed message.
func (i *Info) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ProviderName: i.ProviderName,

		Enabled:  i.Enabled,
		Priority: i.Priority,
		Config:   i.Config,

		HasSecret:        i.HasSecret,
		CredentialSource: i.CredentialSource,
		Usable:           i.Usable,

		UpdatedBy: i.UpdatedBy,

		TMUpdate: i.TMUpdate,
	}
}
