package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/models/recording"
)

// V1DataConfbridgesPost is
// v1 data type request struct for
// /v1/confbridges POST
type V1DataConfbridgesPost struct {
	CustomerID    uuid.UUID                `json:"customer_id,omitempty"`
	ActiveflowID  uuid.UUID                `json:"activeflow_id,omitempty"`
	ReferenceType confbridge.ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID                `json:"reference_id,omitempty"`
	Type          confbridge.Type          `json:"type,omitempty"`
}

// V1DataConfbridgesIDExternalMediaPost is
// v1 data type for
// /v1/confbridges/<confbridge-id>/external-media POST
type V1DataConfbridgesIDExternalMediaPost struct {
	ExternalMediaID uuid.UUID          `json:"external_media_id,omitempty"`
	Type            externalmedia.Type `json:"type,omitempty"`
	ExternalHost    string             `json:"external_host,omitempty"`
	Encapsulation   string    `json:"encapsulation,omitempty"`
	Transport       string    `json:"transport,omitempty"`
	ConnectionType  string    `json:"connection_type,omitempty"`
	Format          string    `json:"format,omitempty"`
}

// V1DataConfbridgesIDRecordingStartPost is
// v1 data type for
// /v1/confbridges/<confbridge-id>/recording_start POST
type V1DataConfbridgesIDRecordingStartPost struct {
	Format       recording.Format `json:"format,omitempty"`         // Format to encode audio in. wav, mp3, ogg
	EndOfSilence int              `json:"end_of_silence,omitempty"` // Maximum duration of silence, in seconds. 0 for no limit.
	EndOfKey     string           `json:"end_of_key,omitempty"`     // DTMF input to terminate recording. none, any, *, #
	Duration     int              `json:"duration,omitempty"`       // Maximum duration of the recording, in seconds. 0 for no limit.
	OnEndFlowID  uuid.UUID        `json:"on_end_flow_id,omitempty"` // Flow ID to execute when the recording ends.
}

// V1DataConfbridgesIDFlagsPost is
// v1 data type for
// /v1/confbridges/<confbridge-id>/flags POST
type V1DataConfbridgesIDFlagsPost struct {
	Flag confbridge.Flag `json:"flag,omitempty"` //
}

// V1DataConfbridgesIDFlagsDelete is
// v1 data type for
// /v1/confbridges/<confbridge-id>/flags DELETE
type V1DataConfbridgesIDFlagsDelete struct {
	Flag confbridge.Flag `json:"flag,omitempty"` //
}
