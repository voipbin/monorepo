package response

import (
	qmqueue "monorepo/bin-queue-manager/models/queue"
)

// BodyQueuesGET is rquest body define for
// GET /v1.0/queues
type BodyQueuesGET struct {
	Result []*qmqueue.WebhookMessage `json:"result"`
	Pagination
}
