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
	TypeAMD Type = "amd"

	// TypeAnswer answers the call.
	// call-manager
	TypeAnswer Type = "answer"

	// TypeBeep plays the beep sound.
	// call-manager
	TypeBeep Type = "beep"

	// TypeBranch gets the variable then execute the correspond action.
	// for example. gets the dtmf input saved variable and jump to the action.
	// flow-manager
	TypeBranch Type = "branch"

	// TypeCall starts a new independent outgoing call with a given flow.
	// it creates a new outgoing call with a new flow.
	// flow-manager
	TypeCall Type = "call"

	// TypeChatbotTalk starts a talk with chatbot.
	// chatbot-manager
	TypeChatbotTalk Type = "chatbot_talk"

	// TypeConditionCallDigits deprecated. use the TypeConditionVariable instead.
	TypeConditionCallDigits Type = "condition_call_digits" // flow-manager. condition check(call's digits)

	// TypeConditionCallStatus deprecated. use the TypeConditionVariable instead.
	TypeConditionCallStatus Type = "condition_call_status" // flow-manager. condition check(call's status)

	// TypeConditionDatetime checks the condition with the current datetime.
	// flow-manager
	TypeConditionDatetime Type = "condition_datetime"

	// TypeConditionVariable checks the condition with the given variable.
	// flow-manager
	TypeConditionVariable Type = "condition_variable"

	// TypeConfbridgeJoin joins the reference to the given confbridge.
	// call-manager
	TypeConfbridgeJoin Type = "confbridge_join"

	// TypeConferenceJoin joins the reference to the given conference.
	// conference-manager
	TypeConferenceJoin Type = "conference_join"

	// TypeConnect creates a new call to the destinations and connects to them.
	// flow-manager
	TypeConnect Type = "connect"

	// conversation_send sends a message to the conversation.
	// conversation-manager
	TypeConversationSend Type = "conversation_send"

	// TypeDigitsReceive receives the digits(dtmfs).
	// call-manager
	TypeDigitsReceive Type = "digits_receive" // call-manager. receive the digits(dtmfs).

	// TypeDigitsSend sends the digits(dtmfs).
	// call-manager
	TypeDigitsSend Type = "digits_send"

	// TypeEcho echo the sound.
	// call-manager
	TypeEcho Type = "echo"

	// TypeExternalMediaStart starts the external media.
	// call-manager
	TypeExternalMediaStart Type = "external_media_start"

	// TypeExternalMediaStop stops the external media.
	// call-manager
	TypeExternalMediaStop Type = "external_media_stop"

	// TypeFetch fetchs the action from the given event url
	// flow-manager
	TypeFetch Type = "fetch"

	// TypeFetchFlow fetchs the flow.
	// flow-manager
	TypeFetchFlow Type = "fetch_flow" // flow-manager.

	// TypeGoto forward the action cursor to the given action id with loop count.
	// flow-manager
	TypeGoto Type = "goto"

	// TypeHangup hangs up the call with a given reason.
	// call-manager
	TypeHangup Type = "hangup"

	// TypeMessageSend sends the SMS to the given destinations.
	// message-manager
	TypeMessageSend Type = "message_send"

	// TypePlay plays the given urls.
	// call-manager
	TypePlay Type = "play"

	// TypeQueueJoin joins the call to the given queue id.
	// flow-manager
	TypeQueueJoin Type = "queue_join"

	// TypeRecordingStart starts the recording.
	// call-manager
	TypeRecordingStart Type = "recording_start"

	// TypeRecordingStop stops the recording.
	// call-manager
	TypeRecordingStop Type = "recording_stop"

	// TypeSleep sleeps the call.
	// call-manager
	TypeSleep Type = "sleep"

	// TypeStop stops the flow.
	// flow-manager
	TypeStop Type = "stop"

	// TypeStreamEcho echo the sound.
	// call-manager
	TypeStreamEcho Type = "stream_echo"

	// TypeTalk generates the audio from the given text(ssml or plain text) and play it.
	// call-manager
	TypeTalk Type = "talk"

	// TypeTranscribeStart starts the transcribe.
	// transcribe-manager
	TypeTranscribeStart Type = "transcribe_start"

	// TypeTranscribeStop stops the transcribe.
	// transcribe-manager
	TypeTranscribeStop Type = "transcribe_stop"

	// TypeTranscribeRecording transcribes the recording file and send it to the webhook.
	// transcribe-manager
	TypeTranscribeRecording Type = "transcribe_recording"

	// TypeVariableSet sets the variable.
	// flow-manager
	TypeVariableSet Type = "variable_set"

	// TypeWebhookSend sends a webhook.
	// webhook-manager
	TypeWebhookSend Type = "webhook_send"
)

// TypeList list of type array
var TypeList []Type = []Type{
	// TypeAgentCall,
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
