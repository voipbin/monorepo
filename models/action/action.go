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
	IDFinish uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000") // reserved action id for finish(hangup)
	IDStart  uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001") // reserved action id for start.
	IDEmpty  uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000002") // empty action
)

// Type type
type Type string

// List of Action types
const (
	TypeAgentCall           Type = "agent_call"           // agent-manager. make a all to the agent.
	TypeAMD                 Type = "amd"                  // call-manager. answering machine detection.
	TypeAnswer              Type = "answer"               // call-manager. answer the call.
	TypeConfbridgeJoin      Type = "confbridge_join"      // call-manager. join to the confbridge.
	TypeConferenceJoin      Type = "conference_join"      // conference-manager. join to the conference.
	TypeConnect             Type = "connect"              // flow-manager. connect to the other destination.
	TypeDTMFReceive         Type = "dtmf_receive"         // call-manager. receive the dtmfs.
	TypeDTMFSend            Type = "dtmf_send"            // call-manager. send the dtmfs.
	TypeEcho                Type = "echo"                 // call-manager.
	TypeExternalMediaStart  Type = "external_media_start" // call-manager.
	TypeExternalMediaStop   Type = "external_media_stop"  // call-manager.
	TypeGoto                Type = "goto"                 // flow-manager.
	TypeHangup              Type = "hangup"               // call-manager.
	TypePatch               Type = "patch"                // flow-manager.
	TypePatchFlow           Type = "patch_flow"           // flow-manager.
	TypePlay                Type = "play"                 // call-manager.
	TypeQueueJoin           Type = "queue_join"           // flow-manager. put the call into the queue.
	TypeRecordingStart      Type = "recording_start"      // call-manager. startr the record of the given call.
	TypeRecordingStop       Type = "recording_stop"       // call-manager. stop the record of the given call.
	TypeSleep               Type = "sleep"                // call-manager. Sleep.
	TypeStreamEcho          Type = "stream_echo"          // call-manager.
	TypeTalk                Type = "talk"                 // call-manager. generate audio from the given text(ssml or plain text) and play it.
	TypeTranscribeStart     Type = "transcribe_start"     // transcribe-manager. start transcribe the call
	TypeTranscribeStop      Type = "transcribe_stop"      // transcribe-manager. stop transcribe the call
	TypeTranscribeRecording Type = "transcribe_recording" // transcribe-manager. transcribe the recording and send it to webhook.
)

// TypeList list of type array
var TypeList []Type = []Type{
	TypeAgentCall,
	TypeAMD,
	TypeAnswer,
	TypeConfbridgeJoin,
	TypeConferenceJoin,
	TypeConnect,
	TypeDTMFReceive,
	TypeDTMFSend,
	TypeEcho,
	TypeExternalMediaStart,
	TypeExternalMediaStop,
	TypeGoto,
	TypeHangup,
	TypePatch,
	TypePatchFlow,
	TypePlay,
	TypeQueueJoin,
	TypeRecordingStart,
	TypeRecordingStop,
	TypeStreamEcho,
	TypeSleep,
	TypeTalk,
	TypeTranscribeStart,
	TypeTranscribeStop,
	TypeTranscribeRecording,
}
