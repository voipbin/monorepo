package response

import (
	cmgroupcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
)

// BodyGroupcallsGET is rquest body define for GET /groupcalls
type BodyGroupcallsGET struct {
	Result []*cmgroupcall.WebhookMessage `json:"result"`
	Pagination
}
