package action

import (
	"github.com/gofrs/uuid"
)

// Action struct
type Action struct {
	ID     uuid.UUID      `json:"id,omitempty"`
	NextID uuid.UUID      `json:"next_id,omitempty"` // represent next target action id. if it not set, just go to next action in the action array.
	Type   Type           `json:"type,omitempty"`
	Option map[string]any `json:"option,omitempty"` // represent the action option. this is used in call-manager, flow-manager, ai-manager, email-manager, message-manager, conversation-manager, webhook-manager.

	TMExecute string `json:"tm_execute,omitempty"` // represent when this action has executed. This is used in call-manager.
}

// list of pre-defined ActionID
var (
	IDEmpty  uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000") // empty action
	IDStart  uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001") // reserved action id for start.
	IDFinish uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000002") // reserved action id for finish
	IDNext   uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000003") // reserved action id for move to next
)

// list of pre-defined actions.
var (
	// ActionFinish is used to represent the finish action.
	// this is used within the activeflow control to move the flow cursor to the finish action
	ActionFinish Action = Action{
		ID: IDFinish,
	}

	// ActionNext is used to represent the next action.
	// this is used within the activeflow control to move the flow cursor to the next action
	ActionNext Action = Action{
		ID: IDNext,
	}

	// ActionEmpty is used to represent an empty action.
	// This is used to halt action execution without moving to the next action.
	ActionEmpty Action = Action{
		ID: IDEmpty,
	}
)

// Type type
type Type string

// List of Action types
const (
	// TypeAMD detects the answering machine.
	// call-manager
	// required media: call
	TypeAMD Type = "amd"

	// TypeAnswer answers the call.
	// call-manager
	// required media: call
	TypeAnswer Type = "answer"

	// TypeAISummary starts a summary with ai.
	// ai-manager
	// required media: none
	TypeAISummary Type = "ai_summary"

	// TypeAITalk starts a talk with ai.
	// ai-manager
	// required media: none
	TypeAITalk Type = "ai_talk"

	// TypeAITask execute the tasks with ai.
	// ai-manager
	// required media: none
	TypeAITask Type = "ai_task"

	// TypeBeep plays the beep sound.
	// call-manager
	// required media: call
	TypeBeep Type = "beep"

	// TypeBlock blocks the action execution until continue to next action request.
	// flow-manager
	// required media: non-rtc
	TypeBlock Type = "block"

	// TypeBranch gets the variable then execute the correspond action.
	// for example. gets the dtmf input saved variable and jump to the action.
	// flow-manager
	// required media: none
	TypeBranch Type = "branch"

	// TypeCall starts a new independent outgoing call with a given flow.
	// it creates a new outgoing call with a new flow.
	// flow-manager
	// required media: none
	TypeCall Type = "call"

	// TypeConditionCallDigits deprecated. use the TypeConditionVariable instead.
	// required media: call
	TypeConditionCallDigits Type = "condition_call_digits" // flow-manager. condition check(call's digits)

	// TypeConditionCallStatus deprecated. use the TypeConditionVariable instead.
	// required media: call
	TypeConditionCallStatus Type = "condition_call_status" // flow-manager. condition check(call's status)

	// TypeConditionDatetime checks the condition with the current datetime.
	// flow-manager
	// required media: none
	TypeConditionDatetime Type = "condition_datetime"

	// TypeConditionVariable checks the condition with the given variable.
	// flow-manager
	// required media: none
	TypeConditionVariable Type = "condition_variable"

	// TypeConfbridgeJoin joins the reference to the given confbridge.
	// call-manager
	// required media: call
	TypeConfbridgeJoin Type = "confbridge_join"

	// TypeConferenceJoin joins the reference to the given conference.
	// conference-manager
	// required media: call
	TypeConferenceJoin Type = "conference_join"

	// TypeConnect creates a new call to the destinations and connects to them.
	// flow-manager
	// required media: call
	TypeConnect Type = "connect"

	// conversation_send sends a message to the conversation.
	// conversation-manager
	// required media: none
	TypeConversationSend Type = "conversation_send"

	// TypeDigitsReceive receives the digits(dtmfs).
	// call-manager
	// required media: call
	TypeDigitsReceive Type = "digits_receive" // call-manager. receive the digits(dtmfs).

	// TypeDigitsSend sends the digits(dtmfs).
	// call-manager
	// required media: call
	TypeDigitsSend Type = "digits_send"

	// TypeEcho echo the sound.
	// call-manager
	// required media: call
	TypeEcho Type = "echo"

	// TypeEmailSend sends the email.
	// email-manager
	// required media: none
	TypeEmailSend Type = "email_send"

	// TypeEmpty is used to represent the empty action.
	// this is used within the activeflow control to move the flow cursor to the empty action
	// required media: none
	TypeEmpty Type = "empty"

	// TypeExternalMediaStart starts the external media.
	// call-manager
	// required media: call
	TypeExternalMediaStart Type = "external_media_start"

	// TypeExternalMediaStop stops the external media.
	// call-manager
	// required media: call
	TypeExternalMediaStop Type = "external_media_stop"

	// TypeFetch fetchs the action from the given event url
	// flow-manager
	// required media: none
	TypeFetch Type = "fetch"

	// TypeFetchFlow fetchs the flow.
	// flow-manager
	// required media: none
	TypeFetchFlow Type = "fetch_flow" // flow-manager.

	// TypeGoto forward the action cursor to the given action id with loop count.
	// flow-manager
	// required media: none
	TypeGoto Type = "goto"

	// TypeHangup hangs up the call with a given reason.
	// call-manager
	// required media: call
	TypeHangup Type = "hangup"

	// TypeMessageSend sends the SMS to the given destinations.
	// message-manager
	// required media: none
	TypeMessageSend Type = "message_send"

	// TypeMute mutes the call
	// call-manager
	// required media: call
	TypeMute Type = "mute"

	// TypePlay plays the given urls.
	// call-manager
	// required media: call
	TypePlay Type = "play"

	// TypeQueueJoin joins the call to the given queue id.
	// flow-manager
	// required media: call
	TypeQueueJoin Type = "queue_join"

	// TypeRecordingStart starts the recording.
	// call-manager
	// required media: call
	TypeRecordingStart Type = "recording_start"

	// TypeRecordingStop stops the recording.
	// call-manager
	// required media: call
	TypeRecordingStop Type = "recording_stop"

	// TypeSleep sleeps the call.
	// call-manager
	// required media: call
	TypeSleep Type = "sleep"

	// TypeStop stops the flow.
	// flow-manager
	// required media: none
	TypeStop Type = "stop"

	// TypeStreamEcho echo the sound.
	// call-manager
	// required media: call
	TypeStreamEcho Type = "stream_echo"

	// TypeTalk generates the audio from the given text(ssml or plain text) and play it.
	// call-manager
	// required media: call
	TypeTalk Type = "talk"

	// TypeTranscribeStart starts the transcribe.
	// transcribe-manager
	// required media: call
	TypeTranscribeStart Type = "transcribe_start"

	// TypeTranscribeStop stops the transcribe.
	// transcribe-manager
	// required media: call
	TypeTranscribeStop Type = "transcribe_stop"

	// TypeTranscribeRecording transcribes the recording file and send it to the webhook.
	// transcribe-manager
	// required media: none
	TypeTranscribeRecording Type = "transcribe_recording"

	// TypeVariableSet sets the variable.
	// flow-manager
	// required media: none
	TypeVariableSet Type = "variable_set"

	// TypeWebhookSend sends a webhook.
	// webhook-manager
	// required media: none
	TypeWebhookSend Type = "webhook_send"
)

// TypeListAll list of type array
var TypeListAll []Type = []Type{
	TypeAMD,
	TypeAnswer,
	TypeAISummary,
	TypeAITalk,
	TypeBeep,
	TypeBlock,
	TypeBranch,
	TypeCall,
	TypeConditionCallDigits,
	TypeConditionCallStatus,
	TypeConditionDatetime,
	TypeConditionVariable,
	TypeConfbridgeJoin,
	TypeConferenceJoin,
	TypeConnect,
	TypeConversationSend,
	TypeDigitsReceive,
	TypeDigitsSend,
	TypeEcho,
	TypeEmailSend,
	TypeExternalMediaStart,
	TypeExternalMediaStop,
	TypeFetch,
	TypeFetchFlow,
	TypeGoto,
	TypeHangup,
	TypeMessageSend,
	TypeMute,
	TypePlay,
	TypeQueueJoin,
	TypeRecordingStart,
	TypeRecordingStop,
	TypeSleep,
	TypeStop,
	TypeStreamEcho,
	TypeTalk,
	TypeTranscribeStart,
	TypeTranscribeStop,
	TypeTranscribeRecording,
	TypeVariableSet,
	TypeWebhookSend,
}

type MediaType string

const (
	MediaTypeNonRealTimeCommunication MediaType = "non_rtc"
	MediaTypeRealTimeCommunication    MediaType = "rtc"
	MediaTypeNone                     MediaType = ""
)

var MapRequiredMediasByType = map[Type][]MediaType{
	TypeAMD:                 {MediaTypeRealTimeCommunication},
	TypeAnswer:              {MediaTypeRealTimeCommunication},
	TypeAISummary:           {MediaTypeNonRealTimeCommunication},
	TypeAITalk:              {MediaTypeNone},
	TypeBeep:                {MediaTypeRealTimeCommunication},
	TypeBlock:               {MediaTypeNonRealTimeCommunication},
	TypeBranch:              {MediaTypeNone},
	TypeCall:                {MediaTypeNone},
	TypeConditionCallDigits: {MediaTypeRealTimeCommunication},
	TypeConditionCallStatus: {MediaTypeRealTimeCommunication},
	TypeConditionDatetime:   {MediaTypeNone},
	TypeConditionVariable:   {MediaTypeNone},
	TypeConfbridgeJoin:      {MediaTypeRealTimeCommunication},
	TypeConferenceJoin:      {MediaTypeRealTimeCommunication},
	TypeConnect:             {MediaTypeRealTimeCommunication},
	TypeConversationSend:    {MediaTypeNone},
	TypeDigitsReceive:       {MediaTypeRealTimeCommunication},
	TypeDigitsSend:          {MediaTypeRealTimeCommunication},
	TypeEcho:                {MediaTypeRealTimeCommunication},
	TypeEmailSend:           {MediaTypeNone},
	TypeEmpty:               {MediaTypeNone},
	TypeExternalMediaStart:  {MediaTypeRealTimeCommunication},
	TypeExternalMediaStop:   {MediaTypeRealTimeCommunication},
	TypeFetch:               {MediaTypeNone},
	TypeFetchFlow:           {MediaTypeNone},
	TypeGoto:                {MediaTypeNone},
	TypeHangup:              {MediaTypeRealTimeCommunication},
	TypeMessageSend:         {MediaTypeNone},
	TypeMute:                {MediaTypeRealTimeCommunication},
	TypePlay:                {MediaTypeRealTimeCommunication},
	TypeQueueJoin:           {MediaTypeRealTimeCommunication},
	TypeRecordingStart:      {MediaTypeRealTimeCommunication},
	TypeRecordingStop:       {MediaTypeRealTimeCommunication},
	TypeSleep:               {MediaTypeRealTimeCommunication},
	TypeStop:                {MediaTypeNone},
	TypeStreamEcho:          {MediaTypeRealTimeCommunication},
	TypeTalk:                {MediaTypeRealTimeCommunication},
	TypeTranscribeStart:     {MediaTypeRealTimeCommunication},
	TypeTranscribeStop:      {MediaTypeRealTimeCommunication},
	TypeTranscribeRecording: {MediaTypeNone},
	TypeVariableSet:         {MediaTypeNone},
	TypeWebhookSend:         {MediaTypeNone},
}
