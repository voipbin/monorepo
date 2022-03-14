package request

import cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

// ParamMessagesGET is request param define for GET /messages
type ParamMessagesGET struct {
	Pagination
}

// BodyMessagesPOST is request param define for POST /messages
type BodyMessagesPOST struct {
	Source       *cmaddress.Address  `json:"source"`
	Destinations []cmaddress.Address `json:"destinations"`
	Text         string              `json:"text"`
}
