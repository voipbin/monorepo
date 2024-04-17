package request

import (
	"github.com/gofrs/uuid"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
)

// V1DataConferencesPost is
// v1 data type request struct for
// /v1/conferences" POST
type V1DataConferencesPost struct {
	Type        conference.Type        `json:"type"`
	CustomerID  uuid.UUID              `json:"customer_id"`
	Name        string                 `json:"name"`
	Detail      string                 `json:"detail"`
	Timeout     int                    `json:"timeout"` // timeout. second
	Data        map[string]interface{} `json:"data"`
	PreActions  []fmaction.Action      `json:"pre_actions"`  // actions before enter the conference.
	PostActions []fmaction.Action      `json:"post_actions"` // actions after leave the conference.
}

// V1DataConferencesIDPut is
// v1 data type request struct for
// /v1/conferences/<conference-id>" PUT
type V1DataConferencesIDPut struct {
	Name        string            `json:"name"`
	Detail      string            `json:"detail"`
	Timeout     int               `json:"timeout"`      // timeout. second
	PreActions  []fmaction.Action `json:"pre_actions"`  // actions before enter the conference.
	PostActions []fmaction.Action `json:"post_actions"` // actions after leave the conference.
}

// V1DataConferencesIDRecordingIDPut is
// v1 data type request struct for
// /v1/conferences/<conference-id>/recording_id" PUT
type V1DataConferencesIDRecordingIDPut struct {
	RecordingID uuid.UUID `json:"recording_id"`
}

// V1DataConferencesIDTranscribeStartPost is
// v1 data type request struct for
// /v1/conferences/<conference-id>/transcribe_start" POST
type V1DataConferencesIDTranscribeStartPost struct {
	Language string `json:"language"`
}
