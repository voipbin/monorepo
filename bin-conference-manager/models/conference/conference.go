package conference

import (
	"fmt"
	"reflect"

	commonidentity "monorepo/bin-common-handler/models/identity"

	uuid "github.com/gofrs/uuid"
)

// Conference type
type Conference struct {
	commonidentity.Identity

	ConfbridgeID uuid.UUID `json:"confbridge_id,omitempty" db:"confbridge_id,uuid"` // confbridge id(call-manager)
	Type         Type      `json:"type,omitempty" db:"type"`

	Status Status `json:"status,omitempty" db:"status"`

	Name    string         `json:"name,omitempty" db:"name"`
	Detail  string         `json:"detail,omitempty" db:"detail"`
	Data    map[string]any `json:"data,omitempty" db:"data,json"`
	Timeout int            `json:"timeout,omitempty" db:"timeout"` // timeout. second

	PreFlowID  uuid.UUID `json:"pre_flow_id,omitempty" db:"pre_flow_id,uuid"`   // pre flow id
	PostFlowID uuid.UUID `json:"post_flow_id,omitempty" db:"post_flow_id,uuid"` // post flow id

	ConferencecallIDs []uuid.UUID `json:"conferencecall_ids,omitempty" db:"conferencecall_ids,json"` // list of conferencecall ids of the conference

	RecordingID  uuid.UUID   `json:"recording_id,omitempty" db:"recording_id,uuid"`
	RecordingIDs []uuid.UUID `json:"recording_ids,omitempty" db:"recording_ids,json"`

	TranscribeID  uuid.UUID   `json:"transcribe_id,omitempty" db:"transcribe_id,uuid"`
	TranscribeIDs []uuid.UUID `json:"transcribe_ids,omitempty" db:"transcribe_ids,json"`

	TMEnd string `json:"tm_end,omitempty" db:"tm_end"` // represent the timestamp for conference ended.

	TMCreate string `json:"tm_create,omitempty" db:"tm_create"`
	TMUpdate string `json:"tm_update,omitempty" db:"tm_update"`
	TMDelete string `json:"tm_delete,omitempty" db:"tm_delete"`
}

// Matches return true if the given items are the same
func (a *Conference) Matches(x any) bool {
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
