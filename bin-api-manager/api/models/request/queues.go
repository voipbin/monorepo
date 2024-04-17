package request

import (
	fmaction "monorepo/bin-flow-manager/models/action"

	qmqueue "monorepo/bin-queue-manager/models/queue"

	"github.com/gofrs/uuid"
)

// ParamQueuesGET is request param define for
// GET /v1.0/queues
type ParamQueuesGET struct {
	Pagination
}

// BodyQueuesPOST is request body define for
// POST /v1.0/queues
type BodyQueuesPOST struct {
	Name           string                `json:"name,omitempty"`
	Detail         string                `json:"detail,omitempty"`
	RoutingMethod  qmqueue.RoutingMethod `json:"routing_method,omitempty"`
	TagIDs         []uuid.UUID           `json:"tag_ids,omitempty"`
	WaitActions    []fmaction.Action     `json:"wait_actions,omitempty"`
	WaitTimeout    int                   `json:"wait_timeout,omitempty"` //
	ServiceTimeout int                   `json:"service_timeout,omitempty"`
}

// BodyQueuesIDPUT is request body define for
// PUT /v1.0/queues/<queue-id>
type BodyQueuesIDPUT struct {
	Name           string                `json:"name,omitempty"`
	Detail         string                `json:"detail,omitempty"`
	RoutingMethod  qmqueue.RoutingMethod `json:"routing_method,omitempty"`
	TagIDs         []uuid.UUID           `json:"tag_ids,omitempty"`
	WaitActions    []fmaction.Action     `json:"wait_actions,omitempty"`
	WaitTimeout    int                   `json:"wait_timeout,omitempty"`
	ServiceTimeout int                   `json:"service_timeout,omitempty"`
}

// BodyQueuesIDTagIDsPUT is request body define for
// PUT /v1.0/queues/<queue-id>/tag_ids
type BodyQueuesIDTagIDsPUT struct {
	TagIDs []uuid.UUID `json:"tag_ids"`
}

// BodyQueuesIDRoutingMethodPUT is request body define for
// PUT /v1.0/queues/<queue-id>/routing_method
type BodyQueuesIDRoutingMethodPUT struct {
	RoutingMethod string `json:"routing_method"`
}

// BodyQueuesIDActionsPUT is request body define for
// PUT /v1.0/queues/<queue-id>/actions
type BodyQueuesIDActionsPUT struct {
	WaitActions    []fmaction.Action `json:"wait_actions"`
	TimeoutWait    int               `json:"timeout_wait"`
	TimeoutService int               `json:"timeout_service"`
}
