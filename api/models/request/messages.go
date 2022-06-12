package request

import commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

// ParamMessagesGET is request param define for GET /messages
type ParamMessagesGET struct {
	Pagination
}

// BodyMessagesPOST is request param define for POST /messages
type BodyMessagesPOST struct {
	Source       *commonaddress.Address  `json:"source"`
	Destinations []commonaddress.Address `json:"destinations"`
	Text         string              `json:"text"`
}
