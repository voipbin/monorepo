package request

import (
	commonaddress "monorepo/bin-common-handler/models/address"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/models/recording"
)

// V1DataCallsPost is
// v1 data type request struct for
// /v1/calls POST
type V1DataCallsPost struct {
	FlowID         uuid.UUID               `json:"flow_id,omitempty"`
	CustomerID     uuid.UUID               `json:"customer_id,omitempty"`
	MasterCallID   uuid.UUID               `json:"master_call_id,omitempty"`
	Source         commonaddress.Address   `json:"source,omitempty"`
	Destinations   []commonaddress.Address `json:"destinations,omitempty"`
	EarlyExecution bool                    `json:"early_execution,omitempty"` // if it sets to true, the call's flow exection will not wait for call answer.
	Connect        bool                    `json:"connect,omitempty"`         // if the call is created for connect, sets this to true,
}

// V1DataCallsIDPost is
// v1 data type request struct for
// /v1/calls/<call-id> POST
type V1DataCallsIDPost struct {
	FlowID         uuid.UUID             `json:"flow_id,omitempty"`
	ActiveflosID   uuid.UUID             `json:"activeflow_id,omitempty"`
	CustomerID     uuid.UUID             `json:"customer_id,omitempty"`
	MasterCallID   uuid.UUID             `json:"master_call_id,omitempty"`
	Source         commonaddress.Address `json:"source,omitempty"`
	Destination    commonaddress.Address `json:"destination,omitempty"`
	GroupcallID    uuid.UUID             `json:"groupcall_id,omitempty"`
	EarlyExecution bool                  `json:"early_execution,omitempty"` // if it sets to true, the call's flow exection will not wait for call answer.
	Connect        bool                  `json:"connect,omitempty"`         // if the call is created for connect, sets this to true,
}

// V1DataCallsIDHealthPost is
// v1 data type request struct for
// CallsIDHealth
// /v1/calls/<call-id>/health-check POST
type V1DataCallsIDHealthPost struct {
	RetryCount int `json:"retry_count,omitempty"`
}

// V1DataCallsIDActionTimeoutPost is
// v1 data type for CallsIDActionTimeout
// /v1/calls/<call-id>/action-timeout POST
type V1DataCallsIDActionTimeoutPost struct {
	ActionID   uuid.UUID     `json:"action_id,omitempty"`
	ActionType fmaction.Type `json:"action_type,omitempty"`
	TMExecute  string        `json:"tm_execute,omitempty"` // represent when this action has executed.
}

// V1DataCallsIDChainedCallIDsPost is
// v1 data type for V1DataCallsIDChainedCallIDsPost
// /v1/calls/<call-id>/chained-call-ids POST
type V1DataCallsIDChainedCallIDsPost struct {
	ChainedCallID uuid.UUID `json:"chained_call_id,omitempty"`
}

// V1DataCallsIDActionNextPost is
// v1 data type for
// /v1/calls/<call-id>/action-next POST
type V1DataCallsIDActionNextPost struct {
	Force bool `json:"force,omitempty"`
}

// V1DataCallsIDExternalMediaPost is
// v1 data type for V1DataCallsIDExternalMediaPost
// /v1/calls/<call-id>/external-media POST
type V1DataCallsIDExternalMediaPost struct {
	ExternalMediaID uuid.UUID               `json:"external_media_id,omitempty"`
	ExternalHost    string                  `json:"external_host,omitempty"`
	Encapsulation   string                  `json:"encapsulation,omitempty"`
	Transport       string                  `json:"transport,omitempty"`
	ConnectionType  string                  `json:"connection_type,omitempty"`
	Format          string                  `json:"format,omitempty"`
	DirectionListen externalmedia.Direction `json:"direction_listen,omitempty"`
	DirectionSpeak  externalmedia.Direction `json:"direction_speak,omitempty"`
}

// V1DataCallsIDDigitsPost is
// v1 data type for V1DataCallsIDDigitsPost
// /v1/calls/<call-id>/digits POST
type V1DataCallsIDDigitsPost struct {
	Digits string `json:"digits,omitempty"`
}

// V1DataCallsIDRecordingIDPut is
// v1 data type for V1DataCallsIDRecordingIDPut
// /v1/calls/<call-id>/recording_id PUT
type V1DataCallsIDRecordingIDPut struct {
	RecordingID uuid.UUID `json:"recording_id,omitempty"`
}

// V1DataCallsIDConfbridgeIDPut is
// v1 data type for
// /v1/calls/<call-id>/confbridge_id PUT
type V1DataCallsIDConfbridgeIDPut struct {
	ConfbridgeID uuid.UUID `json:"confbridge_id,omitempty"`
}

// V1DataCallsIDRecordingStartPost is
// v1 data type for
// /v1/calls/<call-id>/recording_start POST
type V1DataCallsIDRecordingStartPost struct {
	Format       recording.Format `json:"format,omitempty"`         // Format to encode audio in. wav, mp3, ogg
	EndOfSilence int              `json:"end_of_silence,omitempty"` // Maximum duration of silence, in seconds. 0 for no limit.
	EndOfKey     string           `json:"end_of_key,omitempty"`     // DTMF input to terminate recording. none, any, *, #
	Duration     int              `json:"duration,omitempty"`       // Maximum duration of the recording, in seconds. 0 for no limit.
	OnEndFlowID  uuid.UUID        `json:"on_end_flow_id,omitempty"` // Flow to execute when the recording ends.
}

// V1DataCallsIDTalkPost is
// v1 data type for
// /v1/calls/<call-id>/talk POST
type V1DataCallsIDTalkPost struct {
	Text     string `json:"text,omitempty"`     // the text to read(SSML format or plain text)
	Language string `json:"language,omitempty"` // IETF locale-name(ko-KR, en-US)
	Provider string `json:"provider,omitempty"` // tts provider(gcp/aws)
	VoiceID  string `json:"voice_id,omitempty"` // provider-specific voice ID
}

// V1DataCallsIDPlayPost is
// v1 data type for
// /v1/calls/<call-id>/play POST
type V1DataCallsIDPlayPost struct {
	MediaURLs []string `json:"media_urls,omitempty"` // url for media
}

// V1DataCallsIDMutePost is
// v1 data type for
// /v1/calls/<call-id>/mute POST
type V1DataCallsIDMutePost struct {
	Direction call.MuteDirection `json:"direction"`
}

// V1DataCallsIDMuteDelete is
// v1 data type for
// /v1/calls/<call-id>/mute DELETE
type V1DataCallsIDMuteDelete struct {
	Direction call.MuteDirection `json:"direction"`
}
