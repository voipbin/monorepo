package request

import (
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// ParamConferencesGET is rquest param define for
// GET /v1.0/conferences
type ParamConferencesGET struct {
	Pagination
}

// BodyConferencesPOST is rquest body define for
// POST /v1.0/conferences
type BodyConferencesPOST struct {
	Type        cfconference.Type      `json:"type" binding:"required"`
	Name        string                 `json:"name,omitempty"`
	Detail      string                 `json:"detail,omitempty"`
	Timeout     int                    `json:"timeout,omitempty"` // timeout. second
	Data        map[string]interface{} `json:"data,omitempty"`
	PreActions  []fmaction.Action      `json:"pre_actions,omitempty"`
	PostActions []fmaction.Action      `json:"post_actions,omitempty"`
}

// BodyConferencesIDPUT is rquest body define for
// PUT /v1.0/conferences/<conference-id>
type BodyConferencesIDPUT struct {
	Name        string            `json:"name"`
	Detail      string            `json:"detail"`
	Tiemout     int               `json:"timeout"` // seconds
	PreActions  []fmaction.Action `json:"pre_actions"`
	PostActions []fmaction.Action `json:"post_actions"`
}

// BodyConferencesIDTranscribeStartPOST is rquest body define for
// POST /v1.0/conferences/<conference-id>/transcribe_start
type BodyConferencesIDTranscribeStartPOST struct {
	Language string `json:"language"`
}

// ParamConferencesIDMediaStreamGET is rquest param define for
// GET /v1.0/conferences/<conference-id>/media_stream
type ParamConferencesIDMediaStreamGET struct {
	Encapsulation string `form:"encapsulation"`
}
