package confbridge

import "github.com/gofrs/uuid"

// Confbridge type
type Confbridge struct {
	ID uuid.UUID `json:"id"`

	ConferenceID uuid.UUID `json:"conference_id"`
	BridgeID     string    `json:"bridge_id"`

	ChannelCallIDs map[string]uuid.UUID `json:"channel_call_ids"` // channelid:callid

	RecordingID  uuid.UUID   `json:"recording_id"`
	RecordingIDs []uuid.UUID `json:"recording_ids"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}
