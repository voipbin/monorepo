package action

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/gofrs/uuid"
)

// Action struct
type Action struct {
	ID     uuid.UUID       `json:"id"`
	NextID uuid.UUID       `json:"next_id"` // represent next target action id. if it not set, just go to next action in the action array.
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

// list of pre-defined ActionID
var (
	IDEmpty  uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000") // empty action
	IDStart  uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001") // reserved action id for start.
	IDFinish uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000002") // reserved action id for finish
)

// list of pre-defined actions.
var (
	ActionFinish Action = Action{
		ID: IDFinish,
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

	// TypeBeep plays the beep sound.
	// call-manager
	// required media: call
	TypeBeep Type = "beep"

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

	// TypeChatbotTalk starts a talk with chatbot.
	// chatbot-manager
	// required media: call
	TypeChatbotTalk Type = "chatbot_talk"

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
	// required media: call
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
	TypeBeep,
	TypeBranch,
	TypeCall,
	TypeChatbotTalk,
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

// TypeListMediaRequired list of media required action types
var TypeListMediaRequired []Type = []Type{
	TypeAMD,
	TypeAnswer,
	TypeBeep,
	TypeChatbotTalk,
	TypeConditionCallDigits,
	TypeConditionCallStatus,
	TypeConfbridgeJoin,
	TypeConferenceJoin,
	TypeConnect,
	TypeDigitsReceive,
	TypeDigitsSend,
	TypeEcho,
	TypeExternalMediaStart,
	TypeExternalMediaStop,
	TypeHangup,
	TypeMute,
	TypePlay,
	TypeQueueJoin,
	TypeRecordingStart,
	TypeRecordingStop,
	TypeSleep,
	TypeStreamEcho,
	TypeTalk,
	TypeTranscribeStart,
	TypeTranscribeStop,
}
