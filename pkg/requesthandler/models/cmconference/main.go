package cmconference

import (
	uuid "github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/conference"
)

// Conference type
type Conference struct {
	ID       uuid.UUID `json:"id"`
	UserID   uint64    `json:"user_id"`
	Type     Type      `json:"type"`
	BridgeID string    `json:"bridge_id"`

	Status Status `json:"status"`

	Name    string                 `json:"name"`
	Detail  string                 `json:"detail"`
	Data    map[string]interface{} `json:"data"`
	Timeout int                    `json:"timeout"` // timeout. second

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

// Convert returns conference.Conference from cmconference.Conference
func (h *Conference) Convert() *conference.Conference {
	c := &conference.Conference{
		ID:     h.ID,
		UserID: h.UserID,
		Type:   conference.Type(h.Type),

		Status: conference.Status(h.Status),
		Name:   h.Name,
		Detail: h.Detail,

		CallIDs: h.CallIDs,

		RecordingID:  h.RecordingID,
		RecordingIDs: h.RecordingIDs,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}

	return c
}
