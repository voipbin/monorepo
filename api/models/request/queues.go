package request

import (
	"github.com/gofrs/uuid"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// ParamQueuesGET is request param define for GET /queues
type ParamQueuesGET struct {
	Pagination
}

// BodyQueuesPOST is request body define for POST /queues
type BodyQueuesPOST struct {
	Name           string            `json:"name"`
	Detail         string            `json:"detail"`
	RoutingMethod  string            `json:"routing_method"`
	TagIDs         []uuid.UUID       `json:"tag_ids"`
	WaitActions    []fmaction.Action `json:"wait_actions"`
	TimeoutWait    int               `json:"timeout_wait"` //
	TimeoutService int               `json:"timeout_service"`
}

// BodyQueuesIDPUT is request body define for PUT /queues/<queue-id>
type BodyQueuesIDPUT struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// BodyQueuesIDTagIDsPUT is request body define for PUT /queues/<queue-id>/tag_ids
type BodyQueuesIDTagIDsPUT struct {
	TagIDs []uuid.UUID `json:"tag_ids"`
}

// BodyQueuesIDRoutingMethodPUT is request body define for PUT /queues/<queue-id>/routing_method
type BodyQueuesIDRoutingMethodPUT struct {
	RoutingMethod string `json:"routing_method"`
}

// BodyQueuesIDActionsPUT is request body define for PUT /queues/<queue-id>/actions
type BodyQueuesIDActionsPUT struct {
	WaitActions    []fmaction.Action `json:"wait_actions"`
	TimeoutWait    int               `json:"timeout_wait"`
	TimeoutService int               `json:"timeout_service"`
}
