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
	IDBegin uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001") // default action id for call initiating
	IDEnd   uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000") // default action id for call-flow end
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
	TypeAnswer         Type = "answer"
	TypeConferenceJoin Type = "conference_join" // join to the given conference.
	TypeDTMFReceive    Type = "dtmf_receive"    // receive the dtmfs.
	TypeEcho           Type = "echo"            // echo the voice.
	TypeHangup         Type = "hangup"          // call hangup.
	TypePlay           Type = "play"            // play the given file.
	TypeRecordingStart Type = "recording_start" // startr the record of the given call.
	TypeRecordingStop  Type = "recording_stop"  // stop the record of the given call.
	TypeStreamEcho     Type = "stream_echo"     // echo the stream(video/voice) and dtmf.
	TypeTalk           Type = "talk"            // generate audio from the given text(ssml or plain text) and play it.
)

// OptionAnswer defines action answer's option.
type OptionAnswer struct {
	// no option
}

// OptionConferenceJoin defines action conference_join's option.
type OptionConferenceJoin struct {
	ConferenceID string `json:"conference_id"`
}

// OptionDTMFReceive defines action dtmf_receive's option.
type OptionDTMFReceive struct {
	Duration    int    `json:"duration"`       // dtmf receiving duration. ms
	FinishOnKey string `json:"finish_on_key"`  // If set, determines which DTMF triggers the next step. The end key is not included in the resulting variable. If not set, no key will trigger the next step.
	MaxNumKey   int    `json:"max_number_key"` // An optional limit to the number of DTMF events that should be gathered before continuing to the next step.
}

// OptionEcho defines action echo's option.
type OptionEcho struct {
	Duration int  `json:"duration"` // echo duration. ms
	DTMF     bool `json:"dtmf"`     // sending back the dtmf on/off
}

// OptionHangup defines action hangup's option.
type OptionHangup struct {
	// no option
}

// OptionPlay defines action play's option.
type OptionPlay struct {
	StreamURLs []string `json:"stream_urls"` // stream url for media
}

// OptionRecordingStart defines action record's option.
type OptionRecordingStart struct {
	Format       string `json:"format"`         // Format to encode audio in. wav, mp3, ogg
	EndOfSilence int    `json:"end_of_silence"` // Maximum duration of silence, in seconds. 0 for no limit.
	EndOfKey     string `json:"end_of_key"`     // DTMF input to terminate recording. none, any, *, #
	Duration     int    `json:"duration"`       // Maximum duration of the recording, in seconds. 0 for no limit.
	BeepStart    bool   `json:"beep_start"`     // Play beep when recording begins.
}

// OptionRecordingStop defines action record's option.
type OptionRecordingStop struct {
	// no option
}

// OptionStreamEcho defines action stream_echo's option.
type OptionStreamEcho struct {
	Duration int `json:"duration"`
}

// OptionTalk defines action talk's option.
type OptionTalk struct {
	Text     string `json:"text"`     // the text to read(SSML format or plain text)
	Gender   string `json:"gender"`   // gender(male/female/neutral)
	Language string `json:"language"` // IETF locale-name(ko-KR, en-US)
}
