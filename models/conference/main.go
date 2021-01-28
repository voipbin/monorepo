package conference

import uuid "github.com/gofrs/uuid"

// Conference type for client show
type Conference struct {
	ID     uuid.UUID `json:"id"`      // Conference's ID.
	UserID uint64    `json:"user_id"` // Conference owner's ID.
	Type   Type      `json:"type"`    // Conference's type.

	Status Status `json:"status"` // Status.
	Name   string `json:"name"`   // Name.
	Detail string `json:"detail"` // Detail.

	CallIDs []uuid.UUID `json:"call_ids"` // Currently joined call IDs.

	RecordingID  uuid.UUID   `json:"recording_id"`  // Currently recording ID.
	RecordingIDs []uuid.UUID `json:"recording_ids"` // Recorded recording IDs.

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
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
