package action

import (
	"encoding/json"

	"github.com/gofrs/uuid"
)

// Action struct for client show
// used only for the swag.
type Action struct {
	ID     uuid.UUID       `json:"id"`
	NextID uuid.UUID       `json:"next_id"` // represent next target action id. if it not set, just go to next action in the action array.
	Type   Type            `json:"type"`
	Option json.RawMessage `json:"option,omitempty"`

	TMExecute string `json:"tm_execute,omitempty"` // represent when this action has executed. This is used in call-manager.
}

// Type type
type Type string

// List of Action types
const (
	TypeAgentCall           Type = "agent_call"            // agent-manager. make a all to the agent.
	TypeAMD                 Type = "amd"                   // call-manager. answering machine detection.
	TypeAnswer              Type = "answer"                // call-manager. answer the call.
	TypeBeep                Type = "beep"                  // call-manager. play the beep sound.
	TypeBranch              Type = "branch"                // flow-manager. branch the flow
	TypeConditionCallDigits Type = "condition_call_digits" // flow-manager. condition check(call's digits)
	TypeConditionCallStatus Type = "condition_call_status" // flow-manager. condition check(call's status)
	TypeConfbridgeJoin      Type = "confbridge_join"       // call-manager. join to the confbridge.
	TypeConferenceJoin      Type = "conference_join"       // conference-manager. join to the conference.
	TypeConnect             Type = "connect"               // flow-manager. connect to the other destination.
	TypeDigitsReceive       Type = "digits_receive"        // call-manager. receive the digits(dtmfs).
	TypeDigitsSend          Type = "digits_send"           // call-manager. send the digits(dtmfs).
	TypeEcho                Type = "echo"                  // call-manager.
	TypeExternalMediaStart  Type = "external_media_start"  // call-manager.
	TypeExternalMediaStop   Type = "external_media_stop"   // call-manager.
	TypeGoto                Type = "goto"                  // flow-manager.
	TypeHangup              Type = "hangup"                // call-manager.
	TypePatch               Type = "patch"                 // flow-manager.
	TypePatchFlow           Type = "patch_flow"            // flow-manager.
	TypePlay                Type = "play"                  // call-manager.
	TypeQueueJoin           Type = "queue_join"            // flow-manager. put the call into the queue.
	TypeRecordingStart      Type = "recording_start"       // call-manager. startr the record of the given call.
	TypeRecordingStop       Type = "recording_stop"        // call-manager. stop the record of the given call.
	TypeSleep               Type = "sleep"                 // call-manager. Sleep.
	TypeStreamEcho          Type = "stream_echo"           // call-manager.
	TypeTalk                Type = "talk"                  // call-manager. generate audio from the given text(ssml or plain text) and play it.
	TypeTranscribeStart     Type = "transcribe_start"      // transcribe-manager. start transcribe the call
	TypeTranscribeStop      Type = "transcribe_stop"       // transcribe-manager. stop transcribe the call
	TypeTranscribeRecording Type = "transcribe_recording"  // transcribe-manager. transcribe the recording and send it to webhook.
)
