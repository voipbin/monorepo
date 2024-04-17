package request

import commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

// ParamMessagesGET is request param define for
// GET /v1.0/messages
type ParamMessagesGET struct {
	Pagination
}

// BodyMessagesPOST is request param define for
// POST /v1.0/messages
type BodyMessagesPOST struct {
	Source       *commonaddress.Address  `json:"source"`
	Destinations []commonaddress.Address `json:"destinations"`
	Text         string                  `json:"text"`
}
