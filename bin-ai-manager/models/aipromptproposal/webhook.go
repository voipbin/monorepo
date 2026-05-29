package aipromptproposal

import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// WebhookMessage is the externally-visible representation of an AIPromptProposal.
type WebhookMessage struct {
	commonidentity.Identity

	AIID                   uuid.UUID   `json:"ai_id,omitempty"`
	AuditIDs               []uuid.UUID `json:"audit_ids,omitempty"`
	BasisPromptHistoryID   uuid.UUID   `json:"basis_prompt_history_id,omitempty"`
	OriginalPrompt         string      `json:"original_prompt,omitempty"`
	ProposedPrompt         string      `json:"proposed_prompt,omitempty"`
	Rationale              string      `json:"rationale,omitempty"`
	Status                 Status      `json:"status,omitempty"`
	Error                  string      `json:"error,omitempty"`
	AppliedPromptHistoryID uuid.UUID   `json:"applied_prompt_history_id,omitempty"`

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts an AIPromptProposal to its webhook-safe representation.
func (p *AIPromptProposal) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity:               p.Identity,
		AIID:                   p.AIID,
		AuditIDs:               p.AuditIDs,
		BasisPromptHistoryID:   p.BasisPromptHistoryID,
		OriginalPrompt:         p.OriginalPrompt,
		ProposedPrompt:         p.ProposedPrompt,
		Rationale:              p.Rationale,
		Status:                 p.Status,
		Error:                  p.Error,
		AppliedPromptHistoryID: p.AppliedPromptHistoryID,
		TMCreate:               p.TMCreate,
		TMUpdate:               p.TMUpdate,
		TMDelete:               p.TMDelete,
	}
}

// CreateWebhookEvent marshals the WebhookMessage to JSON.
func (p *AIPromptProposal) CreateWebhookEvent() ([]byte, error) {
	e := p.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
