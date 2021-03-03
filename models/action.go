package models

import (
	"encoding/json"

	"github.com/gofrs/uuid"
)

// Action struct for client show
type Action struct {
	ID     uuid.UUID       `json:"id"`
	Type   ActionType      `json:"type"`
	Option json.RawMessage `json:"option,omitempty"`
}

// ActionType type
type ActionType string

// List of Action types
const (
	ActionTypeAnswer         ActionType = "answer"
	ActionTypeConferenceJoin ActionType = "conference_join"
	ActionTypeEcho           ActionType = "echo"
	ActionTypePlay           ActionType = "play"
	ActionTypeStreamEcho     ActionType = "stream_echo"
)

// ActionOptionAnswer defines action answer's option.
type ActionOptionAnswer struct {
	// no option
}

// ActionOptionConferenceJoin defines action conference_join's option.
type ActionOptionConferenceJoin struct {
	ConferenceID string `json:"conference_id"`
}

// ActionOptionEcho struct
type ActionOptionEcho struct {
	Duration int `json:"duration"`
}

// ActionOptionPlay defines action play's option.
type ActionOptionPlay struct {
	StreamURL []string `json:"stream_url"` // stream url for media
}

// ActionOptionStreamEcho defines action stream_echo's option.
type ActionOptionStreamEcho struct {
	Duration int `json:"duration"` // echo duration. ms
}
