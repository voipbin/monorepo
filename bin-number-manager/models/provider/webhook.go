package provider

import (
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-number-manager/models/number"
)

// WebhookMessage is the externally exposed view of a provider setting. It never carries secret
// material: only whether a secret exists (has_secret) and the key version that encrypted it.
type WebhookMessage struct {
	ProviderName number.ProviderName `json:"provider_name"`

	Enabled  bool              `json:"enabled"`
	Priority int               `json:"priority"`
	Config   map[string]string `json:"config,omitempty"`

	HasSecret        bool             `json:"has_secret"`
	KeyVersion       int              `json:"key_version"`
	CredentialSource CredentialSource `json:"credential_source"`
	Usable           bool             `json:"usable"`

	UpdatedBy       *uuid.UUID `json:"updated_by,omitempty"`
	SecretUpdatedBy *uuid.UUID `json:"secret_updated_by,omitempty"`

	TMUpdate       *time.Time `json:"tm_update,omitempty"`
	TMSecretUpdate *time.Time `json:"tm_secret_update,omitempty"`
}

// ConvertWebhookMessage converts the admin view to the externally exposed message.
func (i *Info) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ProviderName: i.ProviderName,

		Enabled:  i.Enabled,
		Priority: i.Priority,
		Config:   i.Config,

		HasSecret:        i.HasSecret,
		KeyVersion:       i.KeyVersion,
		CredentialSource: i.CredentialSource,
		Usable:           i.Usable,

		UpdatedBy:       i.UpdatedBy,
		SecretUpdatedBy: i.SecretUpdatedBy,

		TMUpdate:       i.TMUpdate,
		TMSecretUpdate: i.TMSecretUpdate,
	}
}
