package request

import (
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// ParamConferencesGET is rquest param define for GET /conferences
type ParamConferencesGET struct {
	Pagination
}

// BodyConferencesPOST is rquest body define for POST /conferences
type BodyConferencesPOST struct {
	Type        cfconference.Type `json:"type" binding:"required"`
	Name        string            `json:"name"`
	Detail      string            `json:"detail"`
	PreActions  []fmaction.Action `json:"pre_actions"`
	PostActions []fmaction.Action `json:"post_actions"`
}

// BodyConferencesIDPUT is rquest body define for POST /conferences/<conference-id>
type BodyConferencesIDPUT struct {
	Name        string            `json:"name"`
	Detail      string            `json:"detail"`
	Tiemout     int               `json:"timeout"` // seconds
	PreActions  []fmaction.Action `json:"pre_actions"`
	PostActions []fmaction.Action `json:"post_actions"`
}
