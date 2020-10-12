package action

import (
	"encoding/json"

	"github.com/gofrs/uuid"
)

// Action struct for client show
type Action struct {
	ID     uuid.UUID       `json:"id"`
	Type   Type            `json:"type"`
	Option json.RawMessage `json:"option,omitempty"`
}

// Type type
type Type string

// List of Action types
const (
	TypeAnswer         Type = "answer"
	TypeConferenceJoin Type = "conference_join"
	TypeEcho           Type = "echo"
	TypePlay           Type = "play"
	TypeStreamEcho     Type = "stream_echo"
)

// OptionAnswer defines action answer's option.
type OptionAnswer struct {
	// no option
}

// OptionConferenceJoin defines action conference_join's option.
type OptionConferenceJoin struct {
	ConferenceID string `json:"conference_id"`
}

// OptionEcho struct
type OptionEcho struct {
	Duration int `json:"duration"`
}

// OptionPlay defines action play's option.
type OptionPlay struct {
	StreamURL []string `json:"stream_url"` // stream url for media
}

// OptionStreamEcho defines action stream_echo's option.
type OptionStreamEcho struct {
	Duration int `json:"duration"`
}
