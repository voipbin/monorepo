package response

import (
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
)

// BodyGroupcallsGET is rquest body define for
// GET /v1.0/groupcalls
type BodyGroupcallsGET struct {
	Result []*cmgroupcall.WebhookMessage `json:"result"`
	Pagination
}
