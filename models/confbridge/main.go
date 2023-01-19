package confbridge

import "github.com/gofrs/uuid"

// Confbridge type
type Confbridge struct {
	ID uuid.UUID `json:"id"`

	Type Type `json:"type"`

	BridgeID string `json:"bridge_id"`

	ChannelCallIDs map[string]uuid.UUID `json:"channel_call_ids"` // channelid:callid

	RecordingID     uuid.UUID   `json:"recording_id"`
	RecordingIDs    []uuid.UUID `json:"recording_ids"`
	ExternalMediaID uuid.UUID   `json:"external_media_id"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Type define
type Type string

// list of types
const (
	TypeConnect    Type = "connect"    //
	TypeConference Type = "conference" //
)
