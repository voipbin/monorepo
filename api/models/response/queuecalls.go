package response

import (
	qmqueuecall "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
)

// BodyQueuecallsGET is rquest body define for
// GET /v1.0/queuecalls
type BodyQueuecallsGET struct {
	Result []*qmqueuecall.WebhookMessage `json:"result"`
	Pagination
}
