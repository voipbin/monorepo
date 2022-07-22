package transcribe

import (
	"encoding/json"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID         uuid.UUID `json:"id"`          // Transcribe id
	CustomerID uuid.UUID `json:"customer_id"` // customer

	Type        Type                        `json:"type"`         // type
	ReferenceID uuid.UUID                   `json:"reference_id"` // call/conference/recording's id
	HostID      uuid.UUID                   `json:"host_id"`      // host id
	Language    string                      `json:"language"`     // BCP47 type's language code. en-US
	Transcripts []transcript.WebhookMessage `json:"transcripts"`  // transcripts

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Transcribe) ConvertWebhookMessage() *WebhookMessage {

	transcripts := []transcript.WebhookMessage{}
	for _, t := range h.Transcripts {
		tmp := t.ConvertWebhookMessage()
		transcripts = append(transcripts, *tmp)
	}

	return &WebhookMessage{
		ID:         h.ID,
		CustomerID: h.CustomerID,

		Type:        h.Type,
		ReferenceID: h.ReferenceID,
		HostID:      h.HostID,
		Language:    h.Language,
		Transcripts: transcripts,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generate WebhookEvent
func (h *Transcribe) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
