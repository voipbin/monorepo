package request

import (
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
)

// V1DataConferencesPost is
// v1 data type request struct for
// /v1/conferences" POST
type V1DataConferencesPost struct {
	Type        conference.Type        `json:"type"`
	UserID      uint64                 `json:"user_id"`
	Name        string                 `json:"name"`
	Detail      string                 `json:"detail"`
	Timeout     int                    `json:"timeout"`     // timeout. second
	WebhookURI  string                 `json:"webhook_uri"` // webhook uri
	Data        map[string]interface{} `json:"data"`
	PreActions  []action.Action        `json:"pre_actions"`  // actions before enter the conference.
	PostActions []action.Action        `json:"post_actions"` // actions after leave the conference.
}

// V1DataConferencesIDPut is
// v1 data type request struct for
// /v1/conferences/<conference-id>" PUT
type V1DataConferencesIDPut struct {
	Name        string          `json:"name"`
	Detail      string          `json:"detail"`
	Timeout     int             `json:"timeout"`      // timeout. second
	WebhookURI  string          `json:"webhook_uri"`  // webhook uri
	PreActions  []action.Action `json:"pre_actions"`  // actions before enter the conference.
	PostActions []action.Action `json:"post_actions"` // actions after leave the conference.
}
