package action

import (
	"encoding/json"
	"fmt"
	"reflect"

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

// static ActionID
var (
	IDStart  uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001")
	IDFinish uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000")
)

// Type type
type Type string

// List of Action types
const (
	TypeAnswer              Type = "answer"
	TypeAMD                 Type = "amd"
	TypeConferenceJoin      Type = "conference_join"
	TypeConnect             Type = "connect"
	TypeDTMFReceive         Type = "dtmf_receive" // receive the dtmfs.
	TypeDTMFSend            Type = "dtmf_send"    // send the dtmfs.
	TypeEcho                Type = "echo"
	TypeExternalMediaStart  Type = "external_media_start"
	TypeExternalMediaStop   Type = "external_media_stop"
	TypeHangup              Type = "hangup"
	TypePatch               Type = "patch"
	TypePlay                Type = "play"
	TypeRecordingStart      Type = "recording_start" // startr the record of the given call.
	TypeRecordingStop       Type = "recording_stop"  // stop the record of the given call.
	TypeStreamEcho          Type = "stream_echo"
	TypeTalk                Type = "talk"                 // generate audio from the given text(ssml or plain text) and play it.
	TypeTranscribeStart     Type = "transcribe_start"     // start transcribe the call
	TypeTranscribeStop      Type = "transcribe_stop"      // stop transcribe the call
	TypeTranscribeRecording Type = "transcribe_recording" // transcribe the recording and send it to webhook.
)

// TypeList list of type array
var TypeList []Type = []Type{
	TypeAnswer,
	TypeAMD,
	TypeConferenceJoin,
	TypeConnect,
	TypeDTMFReceive,
	TypeDTMFSend,
	TypeEcho,
	TypeExternalMediaStart,
	TypeExternalMediaStop,
	TypeHangup,
	TypePatch,
	TypePlay,
	TypeRecordingStart,
	TypeRecordingStop,
	TypeStreamEcho,
	TypeTalk,
	TypeTranscribeStart,
	TypeTranscribeStop,
	TypeTranscribeRecording,
}

// OptionAnswer defines action answer's option.
type OptionAnswer struct {
	// no option
}

// OptionAMD defines action amd's option.
type OptionAMD struct {
	MachineHandle bool `json:"machine_handle"` // hangup,delay,continue if the machine answered a call
	Sync          bool `json:"sync"`           // the call flow will be stop until amd done.
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

// OptionDTMFReceive defines action dtmf_receive's option.
type OptionDTMFReceive struct {
	Duration    int    `json:"duration"`       // dtmf receiving duration. ms
	FinishOnKey string `json:"finish_on_key"`  // If set, determines which DTMF triggers the next step. The end key is not included in the resulting variable. If not set, no key will trigger the next step.
	MaxNumKey   int    `json:"max_number_key"` // An optional limit to the number of DTMF events that should be gathered before continuing to the next step.
}

// OptionDTMFSend defines action dtmf_send's option.
type OptionDTMFSend struct {
	DTMFs    string `json:"dtmfs"`    // Keys to send. Allowed set of characters: 0-9, A-D, #, *; with a maximum of 100 keys.
	Duration int    `json:"duration"` // Duration of DTMF tone per key in milliseconds. Allowed values: Between 100 and 1000.
	Interval int    `json:"interval"` // Interval between sending keys in milliseconds. Allowed values: Between 0 and 5000.
}

// OptionEcho struct
type OptionEcho struct {
	Duration int `json:"duration"`
}

// OptionExternalMediaStart defines action OptionExternalMediaStart's option.
type OptionExternalMediaStart struct {
	ExternalHost   string `json:"external_host"`             // external media target host address
	Encapsulation  string `json:"encapsulation,omitempty"`   // encapsulation. default: rtp
	Transport      string `json:"transport,omitempty"`       // transport. default: udp
	ConnectionType string `json:"connection_type,omitempty"` // connection type. default: client
	Format         string `json:"format"`                    // format default: ulaw
	Direction      string `json:"direction,omitempty"`       // direction. default: both
	Data           string `json:"data,omitempty"`            // data
}

// OptionExternalMediaStop defines action OptionExternalMediaStop's option
type OptionExternalMediaStop struct {
	// no option
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

// OptionTranscribeStart defines action TypeTranscribeStart's option.
type OptionTranscribeStart struct {
	Language      string `json:"language"`       // BCP47 format. en-US
	WebhookURI    string `json:"webhook_uri"`    // webhook uri
	WebhookMethod string `json:"webhook_method"` // webhook method
}

// OptionTranscribeStop defines action TypeTranscribeStop's option.
type OptionTranscribeStop struct {
	// no option
}

// OptionTranscribeRecording defines action OptionTranscribeRecording's option.
type OptionTranscribeRecording struct {
	Language      string `json:"language"`       // BCP47 format. en-US
	WebhookURI    string `json:"webhook_uri"`    // webhook uri
	WebhookMethod string `json:"webhook_method"` // webhook method
}
