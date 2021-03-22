package action

import (
	"encoding/json"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
)

// Action struct
type Action struct {
	ID     uuid.UUID       `json:"id"`
	Type   Type            `json:"type"`
	Option json.RawMessage `json:"option,omitempty"`

	TMExecute string `json:"tm_execute,omitempty"` // represent when this action has executed. This is used in call-manager.
}

// static ActionID
var (
	IDStart  uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001")
	IDFinish uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000")
)

// Type type
type Type string

// List of Action types
const (
	TypeAnswer         Type = "answer"
	TypeConferenceJoin Type = "conference_join"
	TypeConnect        Type = "connect"
	TypeDTMFReceive    Type = "dtmf_receive" // receive the dtmfs.
	TypeDTMFSend       Type = "dtmf_send"    // send the dtmfs.
	TypeEcho           Type = "echo"
	TypeHangup         Type = "hangup"
	TypePatch          Type = "patch"
	TypePlay           Type = "play"
	TypeRecordingStart Type = "recording_start" // startr the record of the given call.
	TypeRecordingStop  Type = "recording_stop"  // stop the record of the given call.
	TypeStreamEcho     Type = "stream_echo"
	TypeTalk           Type = "talk" // generate audio from the given text(ssml or plain text) and play it.
)

// OptionAnswer defines action answer's option.
type OptionAnswer struct {
	// no option
}

// OptionConferenceJoin defines action conference_join's option.
type OptionConferenceJoin struct {
	ConferenceID string `json:"conference_id"`
}

// OptionConnect defines action connect's optoin.
type OptionConnect struct {
	Source       address.Address   `json:"source"`       // source infromation.
	Destinations []address.Address `json:"destinations"` // target destinations.
	Unchained    bool              `json:"unchained"`    // If it sets to false, connected destination calls will be hungup when the master call is hangup. Default false.
}

// OptionEcho struct
type OptionEcho struct {
	Duration int `json:"duration"`
}

// OptionHangup defines action hangup's option.
type OptionHangup struct {
	// no option
}

// OptionPatch defines action patch's option.
type OptionPatch struct {
	EventURL    string `json:"event_url"`
	EventMethod string `json:"event_method"`
}

// OptionPlay defines action play's option.
type OptionPlay struct {
	StreamURL []string `json:"stream_url"` // stream url for media
}

// OptionStreamEcho defines action stream_echo's option.
type OptionStreamEcho struct {
	Duration int `json:"duration"`
}
