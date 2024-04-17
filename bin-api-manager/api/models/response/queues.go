package response

import (
	qmqueue "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
)

// BodyQueuesGET is rquest body define for
// GET /v1.0/queues
type BodyQueuesGET struct {
	Result []*qmqueue.WebhookMessage `json:"result"`
	Pagination
}
