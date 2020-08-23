package conference

import (
	uuid "github.com/gofrs/uuid"
)

// Conference type
type Conference struct {
	ID       uuid.UUID `json:"id"`
	Type     Type      `json:"type"`
	BridgeID string    `json:"bridge_id"`

	Status Status `json:"status"`

	Name    string                 `json:"name"`
	Detail  string                 `json:"detail"`
	Data    map[string]interface{} `json:"data"`
	Timeout int                    `json:"timeout"` // timeout. second

	CallIDs []uuid.UUID `json:"call_ids"`

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

// NewConference creates a new conference with given request conference
func NewConference(id uuid.UUID, cType Type, bridgeID string, req *Conference) *Conference {
	cf := &Conference{
		ID:       id,
		Type:     cType,
		BridgeID: bridgeID,

		Name:    req.Name,
		Detail:  req.Detail,
		Data:    req.Data,
		Timeout: req.Timeout,

		CallIDs: []uuid.UUID{},
	}

	return cf
}
