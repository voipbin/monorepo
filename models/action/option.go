package action

import (
	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
)

// OptionAMD defines action amd's option.
type OptionAMD struct {
	MachineHandle string `json:"machine_handle"` // hangup,delay,continue if the machine answered a call
	Async         bool   `json:"async"`          // if it's false, the call flow will be stop until amd done.
}

// OptionAgentCall defines action agent_call's option.
type OptionAgentCall struct {
	AgentID uuid.UUID `json:"agent_id"`
}

// OptionAnswer defines action answer's option.
type OptionAnswer struct {
	// no option
}

// OptionBranch defines action branch's option.
type OptionBranch struct {
	DefaultIndex  int                  `json:"default_index"`  // default index for if the input dtmf does not match any of branch targets. used for forward_id generate
	DefaultID     uuid.UUID            `json:"default_id"`     // default id for if the input dtmf does not match any of branch targets.
	TargetIndexes map[string]int       `json:"target_indexes"` // branch target indexes. used for taget_ids generate
	TargetIDs     map[string]uuid.UUID `json:"target_ids"`     // branch target ids.
}

// OptionConfbridgeJoin defines action confbridge_join's option.
type OptionConfbridgeJoin struct {
	ConfbridgeID uuid.UUID `json:"confbridge_id"`
}

// OptionConferenceJoin defines action conference_join's option.
type OptionConferenceJoin struct {
	ConferenceID uuid.UUID `json:"conference_id"`
}

// OptionConnect defines action connect's optoin.
type OptionConnect struct {
	Source       cmaddress.Address   `json:"source"`       // source infromation.
	Destinations []cmaddress.Address `json:"destinations"` // target destinations.
	Unchained    bool                `json:"unchained"`    // If it sets to false, connected destination calls will be hungup when the master call is hangup. Default false.
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

// OptionGoto defines action goto's option
type OptionGoto struct {
	TargetIndex int       `json:"target_index"` // taget's index of flow aray for go to.
	TargetID    uuid.UUID `json:"target_id"`    // target's action id in the flow array for go to.
	Loop        bool      `json:"loop"`         // if it's true, do the count goto.
	LoopCount   int       `json:"loop_count"`   // loop count.
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

// OptionPatchFlow defines action patch_flow's option.
type OptionPatchFlow struct {
	FlowID uuid.UUID `json:"flow_id"`
}

// OptionPlay defines action play's option.
type OptionPlay struct {
	StreamURLs []string `json:"stream_urls"` // stream url for media
}

// OptionQueueJoin defines action queue_join's option.
type OptionQueueJoin struct {
	QueueID uuid.UUID `json:"queue_id"` // queue's id.
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

// OptionSleep defines action sleep's option.
type OptionSleep struct {
	Duration int `json:"duration"` // Sleep duration. Milliseconds.
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
