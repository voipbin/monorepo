package action

import (
	"encoding/json"

	uuid "github.com/gofrs/uuid"
)

// Action struct
type Action struct {
	ID     uuid.UUID       `json:"id"`
	Type   Type            `json:"type"`
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

// Type type
type Type string

// List of Action types
const (
	TypeEcho   Type = "echo"
	TypeAnswer Type = "answer"
)

// OptionEcho struct
type OptionEcho struct {
	Duration int  `json:"duration"` // echo duration
	DTMF     bool `json:"dtmf"`     // sending back the dtmf on/off
}
