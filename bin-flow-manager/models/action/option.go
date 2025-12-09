package action

import (
	"encoding/json"
	commonaddress "monorepo/bin-common-handler/models/address"

	amaicall "monorepo/bin-ai-manager/models/aicall"
	amsummary "monorepo/bin-ai-manager/models/summary"
	ememail "monorepo/bin-email-manager/models/email"

	"github.com/gofrs/uuid"
)

// ConvertOption converts the option struct to map[string]any.
func ConvertOption(opt any) map[string]any {

	var res map[string]any
	tmp, err := json.Marshal(opt)
	if err != nil {
		return res
	}

	if errUnmarshal := json.Unmarshal(tmp, &res); errUnmarshal != nil {
		return res
	}

	return res
}

// ParseOption parses the option map to the target option struct.
func ParseOption(opt map[string]any, target any) error {
	tmp, err := json.Marshal(opt)
	if err != nil {
		return err
	}

	return json.Unmarshal(tmp, target)
}

// OptionAgentCall defines action agent_call's option.
type OptionAgentCall struct {
	AgentID uuid.UUID `json:"agent_id"` // target agent id.
}

// OptionAISummary defines action ai_summary's option.
type OptionAISummary struct {
	OnEndFlowID   uuid.UUID               `json:"on_end_flow_id,omitempty"` // flow id for the end of recording.
	ReferenceType amsummary.ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID               `json:"reference_id,omitempty"`
	Language      string                  `json:"language,omitempty"`
}

// OptionAITalk defines action ai_talk's option.
type OptionAITalk struct {
	AIID     uuid.UUID       `json:"ai_id,omitempty"`
	Resume   bool            `json:"resume,omitempty"` // resume the previous ai talk.
	Gender   amaicall.Gender `json:"gender,omitempty"`
	Language string          `json:"language,omitempty"` // BCP47 format. en-US
	Duration int             `json:"duration,omitempty"` // ai talk duration. seconds
}

// OptionAITask defines action ai_task's option
type OptionAITask struct {
	AIID uuid.UUID `json:"ai_id,omitempty"`
}

// OptionAMD defines action amd's option.
type OptionAMD struct {
	MachineHandle OptionAMDMachineHandleType `json:"machine_handle,omitempty"` // hangup,continue if the machine answered a call
	Async         bool                       `json:"async,omitempty"`          // if it's false, the call flow will be stop until amd done.
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
	Variable        string               `json:"variable,omitempty"`
	DefaultTargetID uuid.UUID            `json:"default_target_id,omitempty"` // default id for the input dtmf does not match any of branch targets.
	TargetIDs       map[string]uuid.UUID `json:"target_ids,omitempty"`        // branch target ids.
}

// OptionCall defines action call's option.
type OptionCall struct {
	Source         *commonaddress.Address  `json:"source,omitempty"`
	Destinations   []commonaddress.Address `json:"destinations,omitempty"`
	FlowID         uuid.UUID               `json:"flow_id,omitempty"`
	Actions        []Action                `json:"actions,omitempty"`
	Chained        bool                    `json:"chained,omitempty"`         // If it sets to true, the created calls will be hungup when the master call is hangup. Default false.
	EarlyExecution bool                    `json:"early_execution,omitempty"` // if it sets to true, the created call executes the flow(activeflow) before call answer.
}

// OptionConfbridgeJoin defines action confbridge_join's option.
type OptionConfbridgeJoin struct {
	ConfbridgeID uuid.UUID `json:"confbridge_id,omitempty"`
}

// OptionConditionCallDigits defines action condition_call_digits's option.
type OptionConditionCallDigits struct {
	Length int    `json:"length,omitempty"` // digit length for finish
	Key    string `json:"key,omitempty"`    // digit key for finish

	FalseTargetID uuid.UUID `json:"false_target_id,omitempty"` // target id for false case.
}

// OptionConditionCallStatus defines action condition_call_status's option.
type OptionConditionCallStatus struct {
	Status OptionConditionCallStatusStatus `json:"status,omitempty"` // call's status

	FalseTargetID uuid.UUID `json:"false_target_id,omitempty"` // target id for false case.
}

// OptionConditionDatetime defines action condition_datetime's option.
type OptionConditionDatetime struct {
	Condition OptionConditionCommonCondition `json:"condition,omitempty"`

	Minute   int   `json:"minute,omitempty"`   // 0 - 59
	Hour     int   `json:"hour,omitempty"`     // 0 - 23
	Day      int   `json:"day,omitempty"`      // 1 - 31
	Month    int   `json:"month,omitempty"`    // 1 - 12
	Weekdays []int `json:"weekdays,omitempty"` // Sunday: 0, Monday: 1, Tuesday: 2, Wednesday: 3, Thursday: 4, Friday: 5, Saturday: 6

	FalseTargetID uuid.UUID `json:"false_target_id,omitempty"` // target id for false case.
}

// OptionConditionVariable defines action condition_variable's option.
type OptionConditionVariable struct {
	Condition OptionConditionCommonCondition `json:"condition,omitempty"`

	Variable    string                           `json:"variable,omitempty"`   // variable name.
	ValueType   OptionConditionVariableValueType `json:"value_type,omitempty"` // value's type.
	ValueString string                           `json:"value_string,omitempty"`
	ValueNumber float32                          `json:"value_number,omitempty"`
	ValueLength int                              `json:"value_length,omitempty"`

	FalseTargetID uuid.UUID `json:"false_target_id,omitempty"` // target id for false case.
}

// OptionConferenceJoin defines action conference_join's option.
type OptionConferenceJoin struct {
	ConferenceID uuid.UUID `json:"conference_id,omitempty"`
}

// OptionConnect defines action connect's optoin.
type OptionConnect struct {
	// source address.
	Source commonaddress.Address `json:"source,omitempty"`

	// target destination addresses.
	Destinations []commonaddress.Address `json:"destinations,omitempty"`

	// if it sets to true, the call will get early media from the destination.
	EarlyMedia bool `json:"early_media,omitempty"`

	// if it sets to true, the master call will try to the hangup the call with the same reason of the first of the destination calls.
	// this is valid only the first destination call hungup earlier than the master call.
	RelayReason bool `json:"relay_reason,omitempty"`
}

// OptionConversationSend defines action conversation_send's optoin.
type OptionConversationSend struct {
	ConversationID uuid.UUID `json:"conversation_id,omitempty"` // conversation's id.
	Text           string    `json:"text,omitempty"`            // message text.
	Sync           bool      `json:"sync,omitempty"`
}

// OptionDigitsReceive defines action dtmf_receive's option.
type OptionDigitsReceive struct {
	Duration int    `json:"duration,omitempty"` // dtmf receiving duration. ms
	Key      string `json:"key,omitempty"`      // If set, determines which DTMF triggers the next step. The end key is not included in the resulting variable. If not set, no key will trigger the next step.
	Length   int    `json:"length,omitempty"`   // An optional limit to the number of DTMF events that should be gathered before continuing to the next step.
}

// OptionDigitsSend defines action dtmf_send's option.
type OptionDigitsSend struct {
	Digits   string `json:"digits,omitempty"`   // Keys to send. Allowed set of characters: 0-9, A-D, #, *; with a maximum of 100 keys.
	Duration int    `json:"duration,omitempty"` // Duration of DTMF tone per key in milliseconds. Allowed values: Between 100 and 1000.
	Interval int    `json:"interval,omitempty"` // Interval between sending keys in milliseconds. Allowed values: Between 0 and 5000.
}

// OptionEcho struct
type OptionEcho struct {
	Duration int `json:"duration,omitempty"`
}

// OptionEmailSend defines action email_send's option.
type OptionEmailSend struct {
	Destinations []commonaddress.Address `json:"destinations,omitempty"`
	Subject      string                  `json:"subject,omitempty"`
	Content      string                  `json:"content,omitempty"`
	Attachments  []ememail.Attachment    `json:"attachments,omitempty"`
}

// OptionEmpty defines action empty's option.
type OptionEmpty struct {
}

// OptionExternalMediaStart defines action OptionExternalMediaStart's option.
type OptionExternalMediaStart struct {
	ExternalHost    string `json:"external_host,omitempty"`    // external media target host address
	Encapsulation   string `json:"encapsulation,omitempty"`    // encapsulation. default: rtp
	Transport       string `json:"transport,omitempty"`        // transport. default: udp
	ConnectionType  string `json:"connection_type,omitempty"`  // connection type. default: client
	Format          string `json:"format,omitempty"`           // format default: ulaw
	DirectionListen string `json:"direction_listen,omitempty"` // direction. default: ""
	DirectionSpeak  string `json:"direction_speak,omitempty"`  // direction. default: ""
	Data            string `json:"data,omitempty"`             // data
}

// OptionExternalMediaStop defines action OptionExternalMediaStop's option
type OptionExternalMediaStop struct {
	// no option
}

// OptionFetch defines action fetch's option.
type OptionFetch struct {
	EventURL    string `json:"event_url,omitempty"`
	EventMethod string `json:"event_method,omitempty"`
}

// OptionFetchFlow defines action fetch_flow's option.
type OptionFetchFlow struct {
	FlowID uuid.UUID `json:"flow_id,omitempty"`
}

// OptionGoto defines action goto's option
type OptionGoto struct {
	TargetID  uuid.UUID `json:"target_id,omitempty"`  // target's action id in the flow array for go to.
	LoopCount int       `json:"loop_count,omitempty"` // loop count.
}

// OptionHangup defines action hangup's option.
type OptionHangup struct {
	Reason      string    `json:"reason,omitempty"`       // hangup reason code. See detail cmcall.HangupReason
	ReferenceID uuid.UUID `json:"reference_id,omitempty"` // if it's set will hangup the call with the same reason of this referenced call id. This will overwrite the reason option.
}

// OptionMessageSend defines action message_send's option.
type OptionMessageSend struct {
	Source       *commonaddress.Address  `json:"source,omitempty"`
	Destinations []commonaddress.Address `json:"destinations,omitempty"`
	Text         string                  `json:"text,omitempty"`
}

// OptionPlay defines action play's option.
type OptionPlay struct {
	StreamURLs []string `json:"stream_urls,omitempty"` // stream url for media
}

// OptionQueueJoin defines action queue_join's option.
type OptionQueueJoin struct {
	QueueID uuid.UUID `json:"queue_id,omitempty"` // queue's id.
}

// OptionRecordingStart defines action record's option.
type OptionRecordingStart struct {
	Format       string    `json:"format,omitempty"`         // Format to encode audio in. wav, mp3, ogg
	EndOfSilence int       `json:"end_of_silence,omitempty"` // Maximum duration of silence, in seconds. 0 for no limit.
	EndOfKey     string    `json:"end_of_key,omitempty"`     // DTMF input to terminate recording. none, any, *, #
	Duration     int       `json:"duration,omitempty"`       // Maximum duration of the recording, in seconds. 0 for no limit.
	BeepStart    bool      `json:"beep_start,omitempty"`     // Play beep when recording begins.
	OnEndFlowID  uuid.UUID `json:"on_end_flow_id,omitempty"` // flow id for the end of recording.
}

// OptionRecordingStop defines action record's option.
type OptionRecordingStop struct {
	// no option
}

// OptionSleep defines action sleep's option.
type OptionSleep struct {
	Duration int `json:"duration,omitempty"` // Sleep duration. Milliseconds.
}

// OptionStreamEcho defines action stream_echo's option.
type OptionStreamEcho struct {
	Duration int `json:"duration,omitempty"`
}

// OptionTalk defines action talk's option.
type OptionTalk struct {
	Text         string                 `json:"text,omitempty"`          // the text to read(SSML format or plain text)
	Gender       string                 `json:"gender,omitempty"`        // gender(male/female/neutral)
	Language     string                 `json:"language,omitempty"`      // IETF locale-name(ko-KR, en-US)
	DigitsHandle OptionTalkDigitsHandle `json:"digits_handle,omitempty"` // define action when it receives the digits.
	Async        bool                   `json:"async,omitempty"`         // if true, the call flow continues immediately without waiting for talk to complete; if false, the call flow waits until talk is done.
}

// OptionTranscribeStart defines action TypeTranscribeStart's option.
type OptionTranscribeStart struct {
	Language    string    `json:"language,omitempty"`       // BCP47 format. en-US
	OnEndFlowID uuid.UUID `json:"on_end_flow_id,omitempty"` // flow id for the end of recording.
}

// OptionTranscribeStop defines action TypeTranscribeStop's option.
type OptionTranscribeStop struct {
	// no option
}

// OptionTranscribeRecording defines action TypeTranscribeRecording's option.
type OptionTranscribeRecording struct {
	Language    string    `json:"language,omitempty"`       // BCP47 format. en-US
	OnEndFlowID uuid.UUID `json:"on_end_flow_id,omitempty"` // flow id for the end of recording.
}

// OptionVariableSet defines action TypeVariableSet's option.
type OptionVariableSet struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// OptionWebhookSend defines action TypeWebhookSend's option.
type OptionWebhookSend struct {
	Sync     bool   `json:"sync,omitempty"`
	URI      string `json:"uri,omitempty"`
	Method   string `json:"method,omitempty"`    // POST/GET/PUT/DELETE
	DataType string `json:"data_type,omitempty"` // application/json
	Data     string `json:"data,omitempty"`
}
