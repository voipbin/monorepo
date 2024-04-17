package conference

import (
	"fmt"
	"reflect"

	fmaction "monorepo/bin-flow-manager/models/action"

	uuid "github.com/gofrs/uuid"
)

// Conference type
type Conference struct {
	ID           uuid.UUID `json:"id"`
	CustomerID   uuid.UUID `json:"customer_id"`
	ConfbridgeID uuid.UUID `json:"confbridge_id"` // confbridge id(call-manager)
	FlowID       uuid.UUID `json:"flow_id"`       // flow id(flow-manager)
	Type         Type      `json:"type"`

	Status Status `json:"status"`

	Name    string                 `json:"name"`
	Detail  string                 `json:"detail"`
	Data    map[string]interface{} `json:"data"`
	Timeout int                    `json:"timeout"` // timeout. second

	PreActions  []fmaction.Action `json:"pre_actions"`  // pre actions
	PostActions []fmaction.Action `json:"post_actions"` // post actions

	ConferencecallIDs []uuid.UUID `json:"conferencecall_ids"` // list of conferencecall ids of the conference

	RecordingID  uuid.UUID   `json:"recording_id"`
	RecordingIDs []uuid.UUID `json:"recording_ids"`

	TranscribeID  uuid.UUID   `json:"transcribe_id"`
	TranscribeIDs []uuid.UUID `json:"transcribe_ids"`

	TMEnd string `json:"tm_end"` // represent the timestamp for conference ended.

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
	TypeQueue      Type = "queue"      // queue for conference room for queue.
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
