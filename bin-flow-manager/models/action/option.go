package action

import (
	commonaddress "monorepo/bin-common-handler/models/address"

	amaicall "monorepo/bin-ai-manager/models/aicall"
	ememail "monorepo/bin-email-manager/models/email"

	"github.com/gofrs/uuid"
)

// OptionAgentCall defines action agent_call's option.
type OptionAgentCall struct {
	AgentID uuid.UUID `json:"agent_id"` // target agent id.
}

// OptionAITalk defines action ai_talk's option.
type OptionAITalk struct {
	AIID     uuid.UUID       `json:"ai_id"`
	Resume   bool            `json:"resume"` // resume the previous ai talk.
	Gender   amaicall.Gender `json:"gender"`
	Language string          `json:"language"` // BCP47 format. en-US
	Duration int             `json:"duration"` // ai talk duration. seconds
}

// OptionAMD defines action amd's option.
type OptionAMD struct {
	MachineHandle OptionAMDMachineHandleType `json:"machine_handle"` // hangup,continue if the machine answered a call
	Async         bool                       `json:"async"`          // if it's false, the call flow will be stop until amd done.
}

// OptionAnswer defines action answer's option.
type OptionAnswer struct {
	// no option
}

// OptionBeep defines action beep's option.
type OptionBeep struct {
	// no option
}

// OptionBranch defines action branch's option.
type OptionBranch struct {
	Variable        string               `json:"variable"`
	DefaultTargetID uuid.UUID            `json:"default_target_id"` // default id for the input dtmf does not match any of branch targets.
	TargetIDs       map[string]uuid.UUID `json:"target_ids"`        // branch target ids.
}

// OptionCall defines action call's option.
type OptionCall struct {
	Source         *commonaddress.Address  `json:"source"`
	Destinations   []commonaddress.Address `json:"destinations"`
	FlowID         uuid.UUID               `json:"flow_id"`
	Actions        []Action                `json:"actions"`
	Chained        bool                    `json:"chained"`         // If it sets to true, the created calls will be hungup when the master call is hangup. Default false.
	EarlyExecution bool                    `json:"early_execution"` // if it sets to true, the created call executes the flow(activeflow) before call answer.
}

// OptionConfbridgeJoin defines action confbridge_join's option.
type OptionConfbridgeJoin struct {
	ConfbridgeID uuid.UUID `json:"confbridge_id"`
}

// OptionConditionCallDigits defines action condition_call_digits's option.
type OptionConditionCallDigits struct {
	Length int    `json:"length"` // digit length for finish
	Key    string `json:"key"`    // digit key for finish

	FalseTargetID uuid.UUID `json:"false_target_id"` // target id for false case.
}

// OptionConditionCallStatus defines action condition_call_status's option.
type OptionConditionCallStatus struct {
	Status OptionConditionCallStatusStatus `json:"status"` // call's status

	FalseTargetID uuid.UUID `json:"false_target_id"` // target id for false case.
}

// OptionConditionDatetime defines action condition_datetime's option.
type OptionConditionDatetime struct {
	Condition OptionConditionCommonCondition `json:"condition"`

	Minute   int   `json:"minute"`   // 0 - 59
	Hour     int   `json:"hour"`     // 0 - 23
	Day      int   `json:"day"`      // 1 - 31
	Month    int   `json:"month"`    // 1 - 12
	Weekdays []int `json:"weekdays"` // Sunday: 0, Monday: 1, Tuesday: 2, Wednesday: 3, Thursday: 4, Friday: 5, Saturday: 6

	FalseTargetID uuid.UUID `json:"false_target_id"` // target id for false case.
}

// OptionConditionVariable defines action condition_variable's option.
type OptionConditionVariable struct {
	Condition OptionConditionCommonCondition `json:"condition"`

	Variable    string                           `json:"variable"`   // variable name.
	ValueType   OptionConditionVariableValueType `json:"value_type"` // value's type.
	ValueString string                           `json:"value_string"`
	ValueNumber float32                          `json:"value_number"`
	ValueLength int                              `json:"value_length"`

	FalseTargetID uuid.UUID `json:"false_target_id"` // target id for false case.
}

// OptionConferenceJoin defines action conference_join's option.
type OptionConferenceJoin struct {
	ConferenceID uuid.UUID `json:"conference_id"`
}

// OptionConnect defines action connect's optoin.
type OptionConnect struct {
	// source address.
	Source commonaddress.Address `json:"source"`

	// target destination addresses.
	Destinations []commonaddress.Address `json:"destinations"`

	// if it sets to true, the call will get early media from the destination.
	EarlyMedia bool `json:"early_media"`

	// if it sets to true, the master call will try to the hangup the call with the same reason of the first of the destination calls.
	// this is valid only the first destination call hungup earlier than the master call.
	RelayReason bool `json:"relay_reason"`
}

// OptionConversationSend defines action conversation_send's optoin.
type OptionConversationSend struct {
	ConversationID uuid.UUID `json:"conversation_id"` // conversation's id.
	Text           string    `json:"text"`            // message text.
	Sync           bool      `json:"sync"`
}

// OptionDigitsReceive defines action dtmf_receive's option.
type OptionDigitsReceive struct {
	Duration int    `json:"duration"` // dtmf receiving duration. ms
	Key      string `json:"key"`      // If set, determines which DTMF triggers the next step. The end key is not included in the resulting variable. If not set, no key will trigger the next step.
	Length   int    `json:"length"`   // An optional limit to the number of DTMF events that should be gathered before continuing to the next step.
}

// OptionDigitsSend defines action dtmf_send's option.
type OptionDigitsSend struct {
	Digits   string `json:"digits"`   // Keys to send. Allowed set of characters: 0-9, A-D, #, *; with a maximum of 100 keys.
	Duration int    `json:"duration"` // Duration of DTMF tone per key in milliseconds. Allowed values: Between 100 and 1000.
	Interval int    `json:"interval"` // Interval between sending keys in milliseconds. Allowed values: Between 0 and 5000.
}

// OptionEcho struct
type OptionEcho struct {
	Duration int `json:"duration"`
}

// OptionEmailSend defines action email_send's option.
type OptionEmailSend struct {
	Destinations []commonaddress.Address `json:"destinations"`
	Subject      string                  `json:"subject"`
	Content      string                  `json:"content"`
	Attachments  []ememail.Attachment    `json:"attachments,omitempty"`
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

// OptionFetch defines action fetch's option.
type OptionFetch struct {
	EventURL    string `json:"event_url"`
	EventMethod string `json:"event_method"`
}

// OptionFetchFlow defines action fetch_flow's option.
type OptionFetchFlow struct {
	FlowID uuid.UUID `json:"flow_id"`
}

// OptionGoto defines action goto's option
type OptionGoto struct {
	TargetID  uuid.UUID `json:"target_id"`  // target's action id in the flow array for go to.
	LoopCount int       `json:"loop_count"` // loop count.
}

// OptionHangup defines action hangup's option.
type OptionHangup struct {
	Reason      string    `json:"reason"`       // hangup reason code. See detail cmcall.HangupReason
	ReferenceID uuid.UUID `json:"reference_id"` // if it's set will hangup the call with the same reason of this referenced call id. This will overwrite the reason option.
}

// OptionMessageSend defines action message_send's option.
type OptionMessageSend struct {
	Source       *commonaddress.Address  `json:"source"`
	Destinations []commonaddress.Address `json:"destinations"`
	Text         string                  `json:"text"`
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
	Format       string    `json:"format"`         // Format to encode audio in. wav, mp3, ogg
	EndOfSilence int       `json:"end_of_silence"` // Maximum duration of silence, in seconds. 0 for no limit.
	EndOfKey     string    `json:"end_of_key"`     // DTMF input to terminate recording. none, any, *, #
	Duration     int       `json:"duration"`       // Maximum duration of the recording, in seconds. 0 for no limit.
	BeepStart    bool      `json:"beep_start"`     // Play beep when recording begins.
	OnEndFlowID  uuid.UUID `json:"on_end_flow_id"` // flow id for the end of recording.
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
	Text         string                 `json:"text"`          // the text to read(SSML format or plain text)
	Gender       string                 `json:"gender"`        // gender(male/female/neutral)
	Language     string                 `json:"language"`      // IETF locale-name(ko-KR, en-US)
	DigitsHandle OptionTalkDigitsHandle `json:"digits_handle"` // define action when it receives the digits.
}

// OptionTranscribeStart defines action TypeTranscribeStart's option.
type OptionTranscribeStart struct {
	Language    string    `json:"language"`                 // BCP47 format. en-US
	OnEndFlowID uuid.UUID `json:"on_end_flow_id,omitempty"` // flow id for the end of recording.
}

// OptionTranscribeStop defines action TypeTranscribeStop's option.
type OptionTranscribeStop struct {
	// no option
}

// OptionTranscribeRecording defines action TypeTranscribeRecording's option.
type OptionTranscribeRecording struct {
	Language    string    `json:"language"`                 // BCP47 format. en-US
	OnEndFlowID uuid.UUID `json:"on_end_flow_id,omitempty"` // flow id for the end of recording.
}

// OptionVariableSet defines action TypeVariableSet's option.
type OptionVariableSet struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// OptionWebhookSend defines action TypeWebhookSend's option.
type OptionWebhookSend struct {
	Sync     bool   `json:"sync"`
	URI      string `json:"uri"`
	Method   string `json:"method"`    // POST/GET/PUT/DELETE
	DataType string `json:"data_type"` // application/json
	Data     string `json:"data"`
}
