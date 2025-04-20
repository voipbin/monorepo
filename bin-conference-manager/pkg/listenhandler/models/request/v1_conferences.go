package request

import (
	cmrecording "monorepo/bin-call-manager/models/recording"

	"github.com/gofrs/uuid"

	"monorepo/bin-conference-manager/models/conference"
)

// V1DataConferencesPost is
// v1 data type request struct for
// /v1/conferences" POST
type V1DataConferencesPost struct {
	ID         uuid.UUID       `json:"id,omitempty"` // conference id
	CustomerID uuid.UUID       `json:"customer_id,omitempty"`
	Type       conference.Type `json:"type,omitempty"`
	Name       string          `json:"name,omitempty"`
	Detail     string          `json:"detail,omitempty"`
	Data       map[string]any  `json:"data,omitempty"`
	Timeout    int             `json:"timeout,omitempty"`      // timeout. second
	PreFlowID  uuid.UUID       `json:"pre_flow_id,omitempty"`  // pre flow id
	PostFlowID uuid.UUID       `json:"post_flow_id,omitempty"` // post flow id
}

// V1DataConferencesIDPut is
// v1 data type request struct for
// /v1/conferences/<conference-id> PUT
type V1DataConferencesIDPut struct {
	Name       string         `json:"name,omitempty"`
	Detail     string         `json:"detail,omitempty"`
	Data       map[string]any `json:"data,omitempty"`
	Timeout    int            `json:"timeout,omitempty"`      // timeout. second
	PreFlowID  uuid.UUID      `json:"pre_flow_id,omitempty"`  // pre flow id
	PostFlowID uuid.UUID      `json:"post_flow_id,omitempty"` // post flow id
}

// V1DataConferencesPost is
// v1 data type request struct for
// /v1/conferences/<conference-id>/recording_start POST
type V1DataConferencesIDRecordingStartPost struct {
	ActiveflowID uuid.UUID          `json:"activeflow_id,omitempty"`
	Format       cmrecording.Format `json:"format,omitempty"`
	Duration     int                `json:"duration,omitempty"` // duration. second
	OnEndFlowID  uuid.UUID          `json:"on_end_flow_id,omitempty"`
}

// V1DataConferencesIDRecordingIDPut is
// v1 data type request struct for
// /v1/conferences/<conference-id>/recording_id" PUT
type V1DataConferencesIDRecordingIDPut struct {
	RecordingID uuid.UUID `json:"recording_id,omitempty"`
}

// V1DataConferencesIDTranscribeStartPost is
// v1 data type request struct for
// /v1/conferences/<conference-id>/transcribe_start" POST
type V1DataConferencesIDTranscribeStartPost struct {
	Language string `json:"language,omitempty"`
}
