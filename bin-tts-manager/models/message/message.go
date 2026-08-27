package message

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

type Message struct {
	commonidentity.Identity

	StreamingID uuid.UUID `json:"streaming_id,omitempty"` // Current streaming session

	TotalMessage  string `json:"total_message,omitempty"`  // Total message
	PlayedMessage string `json:"played_message,omitempty"` // Played message to be synthesized

	TotalCount  int `json:"total_count,omitempty"` // Total number of times the message should be played
	PlayedCount int `json:"count,omitempty"`       // Number of times the message has been played

	Finish bool `json:"message_finish,omitempty"` // Whether the message has finished playing
}

// EventSubscriptionID returns the subscription address of the global topic exchange
// `bin-manager.event` (VOIP-1404/1405 §2.2). It is the parent StreamingID, not the message's own
// ID.
//
// Message.ID is fixed once, when the vendor streamer is initialized
// (pkg/streaminghandler/gcp.go, aws.go and elevenlabs.go all build the Message from the
// streaming's MessageID at connect time), and it is never refreshed afterwards. From the second
// utterance of the same streaming session on, that captured ID no longer matches
// streaming.MessageID, so it addresses nothing a subscriber could have bound to. The streaming
// session, on the other hand, is stable for the whole session and is the axis every consumer
// follows: `tts-manager.streaming.<id>.#` and `tts-manager.message.<id>.#` then cover one session.
//
// The receiver is a pointer because the event data reaches notifyhandler as a pointer and the
// eventtopic.SubscriptionIdentifier assertion matches the dynamic type.
func (h *Message) EventSubscriptionID() string {
	return h.StreamingID.String()
}
