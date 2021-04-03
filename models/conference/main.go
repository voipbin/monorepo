package conference

import (
	"fmt"
	"reflect"

	uuid "github.com/gofrs/uuid"
)

// Conference type
type Conference struct {
	ID       uuid.UUID `json:"id"`
	UserID   uint64    `json:"user_id"`
	Type     Type      `json:"type"`
	BridgeID string    `json:"bridge_id"` // bridge id for this conference

	Status Status `json:"status"`

	Name    string                 `json:"name"`
	Detail  string                 `json:"detail"`
	Data    map[string]interface{} `json:"data"`
	Timeout int                    `json:"timeout"` // timeout. second

	CallIDs []uuid.UUID `json:"call_ids"` // list of call ids of conference

	RecordingID  uuid.UUID   `json:"recording_id"`
	RecordingIDs []uuid.UUID `json:"recording_ids"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Matches return true if the given items are the same
func (a *Conference) Matches(x interface{}) bool {
	comp := x.(*Conference)
	c := *a

	c.TMCreate = comp.TMCreate

	return reflect.DeepEqual(c, *comp)
}

func (a *Conference) String() string {
	return fmt.Sprintf("%v", *a)
}

// Type conference types
type Type string

// List of conference(bridge) types
const (
	TypeNone       Type = ""
	TypeConference Type = "conference" // conference for more than 3 calls join
	TypeConnect    Type = "connect"    // connect type kicks out the participant if the 1 call has left in the conference.
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

// IsValidConferenceType returns false if the given conference type is not valid
func IsValidConferenceType(confType Type) bool {
	switch confType {
	case TypeNone, TypeConference, TypeConnect:
		return true

	default:
		return false
	}
}

// // NewConference creates a new conference with given request conference
// func NewConference(id uuid.UUID, cType Type, bridgeID string, req *Conference) *Conference {
// 	cf := &Conference{
// 		ID:       id,
// 		Type:     cType,
// 		BridgeID: bridgeID,

// 		UserID:  req.UserID,
// 		Name:    req.Name,
// 		Detail:  req.Detail,
// 		Data:    req.Data,
// 		Timeout: req.Timeout,

// 		CallIDs: []uuid.UUID{},
// 	}

// 	return cf
// }
