package response

import (
	qmqueue "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
)

// BodyQueuesGET is rquest body define for GET /queues
type BodyQueuesGET struct {
	Result []*qmqueue.Event `json:"result"`
	Pagination
}
