package streaming

import (
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-transcribe-manager/models/transcript"
)

// Speech represents a speech recognition event from a streaming session.
// This is the internal struct passed to PublishWebhookEvent.
// PublishEvent serializes all fields (including Language) for the internal queue.
// PublishWebhook calls CreateWebhookEvent() which filters to WebhookMessage.
type Speech struct {
	commonidentity.Identity

	StreamingID  uuid.UUID           `json:"streaming_id"`
	TranscribeID uuid.UUID           `json:"transcribe_id"`
	Language     string              `json:"language"`
	Direction    transcript.Direction `json:"direction"`

	Message string     `json:"message,omitempty"`
	TMEvent *time.Time `json:"tm_event"`

	TMCreate *time.Time `json:"tm_create"`
}
