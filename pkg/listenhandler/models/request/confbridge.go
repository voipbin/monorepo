package request

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
)

// V1DataConfbridgesPost is
// v1 data type request struct for
// /v1/confbridges POST
type V1DataConfbridgesPost struct {
	CustomerID uuid.UUID       `json:"customer_id"`
	Type       confbridge.Type `json:"type"`
}

// V1DataConfbridgesIDExternalMediaPost is
// v1 data type for
// /v1/confbridges/<confbridge-id>/external-media POST
type V1DataConfbridgesIDExternalMediaPost struct {
	ExternalHost   string `json:"external_host"`
	Encapsulation  string `json:"encapsulation"`
	Transport      string `json:"transport"`
	ConnectionType string `json:"connection_type"`
	Format         string `json:"format"`
	Direction      string `json:"direction"`
}

// V1DataConfbridgesIDRecordingStartPost is
// v1 data type for
// /v1/confbridges/<confbridge-id>/recording_start POST
type V1DataConfbridgesIDRecordingStartPost struct {
	Format       recording.Format `json:"format"`         // Format to encode audio in. wav, mp3, ogg
	EndOfSilence int              `json:"end_of_silence"` // Maximum duration of silence, in seconds. 0 for no limit.
	EndOfKey     string           `json:"end_of_key"`     // DTMF input to terminate recording. none, any, *, #
	Duration     int              `json:"duration"`       // Maximum duration of the recording, in seconds. 0 for no limit.
}
