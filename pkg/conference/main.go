package conference

import uuid "github.com/gofrs/uuid"

// Conference type
type Conference struct {
	ID       uuid.UUID
	Type     Type
	BridgeID string

	Status Status
	Name   string
	Detail string
	Data   []map[string]interface{}

	BridgeIDs []string
	CallIDs   []uuid.UUID

	TMCreate string
	TMUpdate string
	TMDelete string
}

// Type conference types
type Type string

// List of conference(bridge) types
const (
	TypeNone       Type = ""
	TypeEcho       Type = "echo"       // conference for echoing.
	TypeTransfer   Type = "transfer"   // conference for simple transfer
	TypeConference Type = "conference" // conference for more than 3 calls join
)

// Status type
type Status string

// List of Status types
const (
	StatusStarting    Status = "starting"
	StatusProgressing Status = "progressing"
	StatusStopping    Status = "stopping"
	StatusTerminated  Status = "terminated"
)

// NewConference creates a new conference
func NewConference(id uuid.UUID, cType Type, bridgeID, name, detail string) *Conference {
	cf := &Conference{
		ID:       id,
		Type:     cType,
		BridgeID: bridgeID,

		Name:   name,
		Detail: detail,

		BridgeIDs: []string{},
		CallIDs:   []uuid.UUID{},
	}

	return cf
}
