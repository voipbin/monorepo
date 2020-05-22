package action

import (
	"encoding/json"
	"fmt"
	"reflect"

	uuid "github.com/gofrs/uuid"
)

// Action struct
type Action struct {
	ID     uuid.UUID       `json:"id"`
	Type   Type            `json:"type"`
	Option json.RawMessage `json:"option,omitempty"`
	Next   uuid.UUID       `json:"next"` // represent next action's next action.

	TMExecute string `json:"tm_execute"` // represent when this action has executed.
}

// Flow struct
type Flow struct {
	ID       uuid.UUID `json:"id"`
	Revision uuid.UUID `json:"revision"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Actions []Action `json:"actions"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Predefined special IDs
var (
	IDBegin uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001")
	IDEnd   uuid.UUID = uuid.Nil
)

// Matches return true if the given items are the same
func (a *Action) Matches(x interface{}) bool {
	compAction := x.(*Action)
	act := *a
	act.TMExecute = compAction.TMExecute
	return reflect.DeepEqual(&act, compAction)
}

func (a *Action) String() string {
	return fmt.Sprintf("%v", *a)
}

// Type type
type Type string

// List of Action types
const (
	TypeEcho           Type = "echo"
	TypeAnswer         Type = "answer"
	TypeConferenceJoin Type = "conference_join"
)

// OptionEcho struct
type OptionEcho struct {
	Duration int  `json:"duration"` // echo duration. ms
	DTMF     bool `json:"dtmf"`     // sending back the dtmf on/off
}

// OptionConferenceJoin struct
type OptionConferenceJoin struct {
	ConferenceID string `json:"conference_id"`
}
