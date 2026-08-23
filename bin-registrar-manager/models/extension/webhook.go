package extension

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

// WebhookMessage defines
type WebhookMessage struct {
	commonidentity.Identity

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Extension string `json:"extension"`

	DomainName string `json:"domain_name"` // full SIP domain (realm) of the extension. e.g. ab12.reg.voipbin.net
	Username   string `json:"username"`    // same as the Extension. This used by the kamailio's INVITE validation
	Password   string `json:"password"`

	DirectHash string `json:"direct_hash"`

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Extension) ConvertWebhookMessage() *WebhookMessage {
	// SERVING RULE (VOIP-1385): the exposed domain_name is the full realm.
	// Serve Realm when set; when Realm is empty (legacy pre-Feb-2024 rows) the
	// stored DomainName column holds a bare customer uuid until the migration
	// batch rewrites it — registrar-manager's read path already normalizes it
	// to the computed legacy realm before the model leaves the service, so the
	// fallback below never serves the raw bare uuid.
	domainName := h.DomainName
	if h.Realm != "" {
		domainName = h.Realm
	}

	return &WebhookMessage{
		Identity: h.Identity,

		Name:   h.Name,
		Detail: h.Detail,

		Extension: h.Extension,

		DomainName: domainName,
		Username:   h.Username,
		Password:   h.Password,

		DirectHash: h.DirectHash,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Extension) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
