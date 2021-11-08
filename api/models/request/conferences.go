package request

import (
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/conference"
)

// ParamConferencesGET is rquest param define for GET /conferences
type ParamConferencesGET struct {
	Pagination
}

// BodyConferencesPOST is rquest body define for POST /conferences
type BodyConferencesPOST struct {
	Type        conference.Type `json:"type" binding:"required"`
	Name        string          `json:"name"`
	Detail      string          `json:"detail"`
	WebhookURI  string          `json:"webhook_uri"`
	PreActions  []action.Action `json:"pre_actions"`
	PostActions []action.Action `json:"post_actions"`
}
