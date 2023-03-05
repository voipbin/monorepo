package request

import (
	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
)

// V1DataCallsPost is
// v1 data type request struct for
// /v1/calls POST
type V1DataCallsPost struct {
	FlowID                    uuid.UUID               `json:"flow_id"`
	CustomerID                uuid.UUID               `json:"customer_id"`
	MasterCallID              uuid.UUID               `json:"master_call_id"`
	Source                    commonaddress.Address   `json:"source"`
	Destinations              []commonaddress.Address `json:"destinations"`
	EarlyExecution            bool                    `json:"early_execution"`               // if it sets to true, the call's flow exection will not wait for call answer.
	Connect                   bool                    `json:"connect"`                       // if the call is created for connect, sets this to true,
	ExecuteNextMasterOnHangup bool                    `json:"execute_next_master_on_hangup"` // deprecated. if it sets to true, the master call will execute the next action when the outgoing call hangup with not normal.
}

// V1DataCallsIDPost is
// v1 data type request struct for
// /v1/calls/<call-id> POST
type V1DataCallsIDPost struct {
	FlowID                    uuid.UUID             `json:"flow_id"`
	ActiveflosID              uuid.UUID             `json:"activeflow_id"`
	CustomerID                uuid.UUID             `json:"customer_id"`
	MasterCallID              uuid.UUID             `json:"master_call_id"`
	Source                    commonaddress.Address `json:"source"`
	Destination               commonaddress.Address `json:"destination"`
	GroupcallID               uuid.UUID             `json:"groupcall_id"`
	EarlyExecution            bool                  `json:"early_execution"`               // if it sets to true, the call's flow exection will not wait for call answer.
	Connect                   bool                  `json:"connect"`                       // if the call is created for connect, sets this to true,
	ExecuteNextMasterOnHangup bool                  `json:"execute_next_master_on_hangup"` // deprecated. if it sets to true, the master call will execute the next action when the outgoing call hangup with not normal.
}

// V1DataCallsIDHealthPost is
// v1 data type request struct for
// CallsIDHealth
// /v1/calls/<call-id>/health-check POST
type V1DataCallsIDHealthPost struct {
	RetryCount int `json:"retry_count"`
	Delay      int `json:"delay"`
}

// V1DataCallsIDActionTimeoutPost is
// v1 data type for CallsIDActionTimeout
// /v1/calls/<call-id>/action-timeout POST
type V1DataCallsIDActionTimeoutPost struct {
	ActionID   uuid.UUID     `json:"action_id"`
	ActionType fmaction.Type `json:"action_type"`
	TMExecute  string        `json:"tm_execute"` // represent when this action has executed.
}

// V1DataCallsIDChainedCallIDsPost is
// v1 data type for V1DataCallsIDChainedCallIDsPost
// /v1/calls/<call-id>/chained-call-ids POST
type V1DataCallsIDChainedCallIDsPost struct {
	ChainedCallID uuid.UUID `json:"chained_call_id"`
}

// V1DataCallsIDActionNextPost is
// v1 data type for
// /v1/calls/<call-id>/action-next POST
type V1DataCallsIDActionNextPost struct {
	Force bool `json:"force"`
}

// V1DataCallsIDExternalMediaPost is
// v1 data type for V1DataCallsIDExternalMediaPost
// /v1/calls/<call-id>/external-media POST
type V1DataCallsIDExternalMediaPost struct {
	ExternalHost   string `json:"external_host"`
	Encapsulation  string `json:"encapsulation"`
	Transport      string `json:"transport"`
	ConnectionType string `json:"connection_type"`
	Format         string `json:"format"`
	Direction      string `json:"direction"`
}

// V1DataCallsIDDigitsPost is
// v1 data type for V1DataCallsIDDigitsPost
// /v1/calls/<call-id>/digits POST
type V1DataCallsIDDigitsPost struct {
	Digits string `json:"digits"`
}

// V1DataCallsIDRecordingIDPut is
// v1 data type for V1DataCallsIDRecordingIDPut
// /v1/calls/<call-id>/recording_id PUT
type V1DataCallsIDRecordingIDPut struct {
	RecordingID uuid.UUID `json:"recording_id"`
}

// V1DataCallsIDConfbridgeIDPut is
// v1 data type for
// /v1/calls/<call-id>/confbridge_id PUT
type V1DataCallsIDConfbridgeIDPut struct {
	ConfbridgeID uuid.UUID `json:"confbridge_id"`
}

// V1DataCallsIDRecordingStartPost is
// v1 data type for
// /v1/calls/<call-id>/recording_start POST
type V1DataCallsIDRecordingStartPost struct {
	Format       recording.Format `json:"format"`         // Format to encode audio in. wav, mp3, ogg
	EndOfSilence int              `json:"end_of_silence"` // Maximum duration of silence, in seconds. 0 for no limit.
	EndOfKey     string           `json:"end_of_key"`     // DTMF input to terminate recording. none, any, *, #
	Duration     int              `json:"duration"`       // Maximum duration of the recording, in seconds. 0 for no limit.
}

// V1DataCallsIDTalkPost is
// v1 data type for
// /v1/calls/<call-id>/talk POST
type V1DataCallsIDTalkPost struct {
	Text     string `json:"text"`     // the text to read(SSML format or plain text)
	Gender   string `json:"gender"`   // gender(male/female/neutral)
	Language string `json:"language"` // IETF locale-name(ko-KR, en-US)
}

// V1DataCallsIDPlayPost is
// v1 data type for
// /v1/calls/<call-id>/play POST
type V1DataCallsIDPlayPost struct {
	MediaURLs []string `json:"media_urls"` // url for media
}
