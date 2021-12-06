package response

import "gitlab.com/voipbin/bin-manager/api-manager.git/models/call"

// BodyCallsGET is response body define for GET /calls
type BodyCallsGET struct {
	Result []*call.Call `json:"result"`
	Pagination
}
