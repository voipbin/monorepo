package conference

import uuid "github.com/gofrs/uuid"

// Conference type for client show
type Conference struct {
	ID     uuid.UUID `json:"id"`
	UserID uint64    `json:"user_id"`
	Type   Type      `json:"type"`

	Status Status `json:"status"`
	Name   string `json:"name"`
	Detail string `json:"detail"`

	CallIDs []uuid.UUID `json:"call_ids"`

	RecordingID  string   `json:"recording_id"`
	RecordingIDs []string `json:"recording_ids"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Type conference types
type Type string

// List of conference(bridge) types
const (
	TypeNone       Type = ""
	TypeConference Type = "conference" // conference for more than 3 calls join
)

// Status type
type Status string

// List of Status types
const (
	StatusStarting    Status = "starting"
	StatusProgressing Status = "progressing"
	StatusTerminating Status = "terminating"
	StatusTerminated  Status = "terminated"
)
