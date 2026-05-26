package aiaudit

import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// WebhookMessage is the externally-visible representation of an AIAudit.
type WebhookMessage struct {
	commonidentity.Identity

	AIcallID        uuid.UUID       `json:"aicall_id,omitempty"`
	AIID            uuid.UUID       `json:"ai_id,omitempty"`
	PromptHistoryID uuid.UUID       `json:"prompt_history_id,omitempty"`
	Status          Status          `json:"status,omitempty"`
	OverallScore    *int            `json:"overall_score"`
	Evaluation      json.RawMessage `json:"evaluation"`
	Language        string          `json:"language,omitempty"`
	Error           string          `json:"error,omitempty"`

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts an AIAudit to its webhook-safe representation.
func (a *AIAudit) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity:        a.Identity,
		AIcallID:        a.AIcallID,
		AIID:            a.AIID,
		PromptHistoryID: a.PromptHistoryID,
		Status:          a.Status,
		OverallScore:    a.OverallScore,
		Evaluation:      a.Evaluation,
		Language:        a.Language,
		Error:           a.Error,
		TMCreate:        a.TMCreate,
		TMUpdate:        a.TMUpdate,
		TMDelete:        a.TMDelete,
	}
}

// CreateWebhookEvent marshals the WebhookMessage to JSON.
func (a *AIAudit) CreateWebhookEvent() ([]byte, error) {
	e := a.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
