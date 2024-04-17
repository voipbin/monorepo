package response

import (
	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"
)

// BodyQueuecallsGET is rquest body define for
// GET /v1.0/queuecalls
type BodyQueuecallsGET struct {
	Result []*qmqueuecall.WebhookMessage `json:"result"`
	Pagination
}
