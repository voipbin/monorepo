package request

import "gitlab.com/voipbin/bin-manager/call-manager.git/models/conference"

// V1DataConferencesIDPost is
// v1 data type request struct for
// /v1/conferences/<id>" POST
type V1DataConferencesIDPost struct {
	Type    conference.Type        `json:"type"`
	UserID  uint64                 `json:"user_id"`
	Name    string                 `json:"name"`
	Detail  string                 `json:"detail"`
	Timeout int                    `json:"timeout"` // timeout. second
	Data    map[string]interface{} `json:"data"`
}
