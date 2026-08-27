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

// EventSubscriptionID returns the subscription address of the global topic exchange
// `bin-manager.event` (VOIP-1404). It is the parent TranscribeID, not the speech's own ID:
// NewSpeech generates a fresh random ID for every single event, so the own ID is not an address
// anybody could bind to in advance. Subscribers follow one transcription session, and every event
// of that session carries the same transcribe-id.
//
// The receiver is a pointer because the event data reaches notifyhandler as a pointer and the
// eventtopic.SubscriptionIdentifier assertion matches the dynamic type.
func (h *Speech) EventSubscriptionID() string {
	return h.TranscribeID.String()
}
