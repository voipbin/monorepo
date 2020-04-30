package flow

import (
	"encoding/json"

	"github.com/gofrs/uuid"
)

// static ActionID
var (
	ActionIDStart  uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001")
	ActionIDFinish uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000")
)

// ????
var (
	FlowRevisionLatest uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001")
)

// Action struct
type Action struct {
	ID     uuid.UUID       `json:"id"`
	Type   ActionType      `json:"type"`
	Option json.RawMessage `json:"option,omitempty"`
	Next   uuid.UUID       `json:"next"`
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

// ActionType type
type ActionType string

// List of Action types
const (
	ActionTypeEcho ActionType = "echo"
)

// ActionOptionEcho struct
type ActionOptionEcho struct {
	Duration int `json:"duration"`
}
