package request

import (
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"

	"monorepo/bin-queue-manager/models/queue"
)

// V1DataQueuesPost is
// v1 data type request struct for
// /v1/queues POST
type V1DataQueuesPost struct {
	CustomerID uuid.UUID `json:"customer_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	RoutingMethod string      `json:"routing_method"`
	TagIDs        []uuid.UUID `json:"tag_ids"`

	WaitActions    []fmaction.Action `json:"wait_actions"`
	WaitTimeout    int               `json:"wait_timeout"`    // wait timeout(ms)
	ServiceTimeout int               `json:"service_timeout"` // service timeout(ms)
}

// V1DataQueuesIDPut is
// v1 data type request struct for
// /v1/queues/<queue-id> PUT
type V1DataQueuesIDPut struct {
	Name           string              `json:"name"`
	Detail         string              `json:"detail"`
	RoutingMethod  queue.RoutingMethod `json:"routing_method,omitempty"`
	TagIDs         []uuid.UUID         `json:"tag_ids,omitempty"`
	WaitActions    []fmaction.Action   `json:"wait_actions,omitempty"`
	WaitTimeout    int                 `json:"wait_timeout,omitempty"`
	ServiceTimeout int                 `json:"service_timeout,omitempty"`
}

// V1DataQueuesIDQueuecallsPost is
// v1 data type request struct for
// /v1/queues/<queue-id>/queuecalls POST
type V1DataQueuesIDQueuecallsPost struct {
	ReferenceType         string    `json:"reference_type"`
	ReferenceID           uuid.UUID `json:"reference_id"`
	ReferenceActiveflowID uuid.UUID `json:"reference_activeflow_id"`
	ExitActionID          uuid.UUID `json:"exit_action_id"`
}

// V1DataQueuesIDTagIDsPut is
// v1 data type request struct for
// /v1/queues/<queue-id>/tag_ids PUT
type V1DataQueuesIDTagIDsPut struct {
	TagIDs []uuid.UUID `json:"tag_ids"`
}

// V1DataQueuesIDRoutingMethodPut is
// v1 data type request struct for
// /v1/queues/<queue-id>/routing_method PUT
type V1DataQueuesIDRoutingMethodPut struct {
	RoutingMethod string `json:"routing_method"`
}

// V1DataQueuesIDWaitActionsPut is
// v1 data type request struct for
// /v1/queues/<queue-id>/wait_actions PUT
type V1DataQueuesIDWaitActionsPut struct {
	WaitActions    []fmaction.Action `json:"wait_actions"`
	WaitTimeout    int               `json:"wait_timeout"`
	ServiceTimeout int               `json:"service_timeout"`
}

// V1DataQueuesIDExecutePut is
// v1 data type request struct for
// /v1/queues/<queue-id>/execute PUT
type V1DataQueuesIDExecutePut struct {
	Execute queue.Execute `json:"execute"`
}
