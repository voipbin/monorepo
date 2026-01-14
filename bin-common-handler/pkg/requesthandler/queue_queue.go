package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"

	qmqueue "monorepo/bin-queue-manager/models/queue"
	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"
	qmrequest "monorepo/bin-queue-manager/pkg/listenhandler/models/request"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// QueueV1QueueGets sends a request to queue-manager
// to get a list of queues.
// Returns list of queues
func (r *requestHandler) QueueV1QueueGets(ctx context.Context, pageToken string, pageSize uint64, filters map[qmqueue.Field]any) ([]qmqueue.Queue, error) {
	uri := fmt.Sprintf("/v1/queues?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodGet, "queue/queues", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []qmqueue.Queue
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// QueueV1QueueGet sends a request to queue-manager
// to getting the queue.
// it returns an queue if it succeed.
func (r *requestHandler) QueueV1QueueGet(ctx context.Context, queueID uuid.UUID) (*qmqueue.Queue, error) {
	uri := fmt.Sprintf("/v1/queues/%s", queueID)

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodGet, "queue/queues", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res qmqueue.Queue
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// QueueV1QueueCreate sends the request to create the queue.
func (r *requestHandler) QueueV1QueueCreate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	routingMethod qmqueue.RoutingMethod,
	tagIDs []uuid.UUID,
	waitFlowID uuid.UUID,
	timeoutWait int,
	timeoutService int,
) (*qmqueue.Queue, error) {
	uri := "/v1/queues"

	data := &qmrequest.V1DataQueuesPost{
		CustomerID:     customerID,
		Name:           name,
		Detail:         detail,
		RoutingMethod:  string(routingMethod),
		TagIDs:         tagIDs,
		WaitFlowID:     waitFlowID,
		WaitTimeout:    timeoutWait,
		ServiceTimeout: timeoutService,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPost, "queue/queues", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res qmqueue.Queue
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// QueueV1QueueDelete sends a request to queue-manager
// to deleteing the queue.
// it returns an error if it failed.
func (r *requestHandler) QueueV1QueueDelete(ctx context.Context, queueID uuid.UUID) (*qmqueue.Queue, error) {
	uri := fmt.Sprintf("/v1/queues/%s", queueID)

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodDelete, "queue/queues", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res qmqueue.Queue
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// QueueV1QueueUpdate sends the request to update the queue.
func (r *requestHandler) QueueV1QueueUpdate(
	ctx context.Context,
	queueID uuid.UUID,
	name string,
	detail string,
	routingMethod qmqueue.RoutingMethod,
	tagIDs []uuid.UUID,
	waitFlowID uuid.UUID,
	waitTimeout int,
	serviceTimeout int,
) (*qmqueue.Queue, error) {
	uri := fmt.Sprintf("/v1/queues/%s", queueID)

	data := &qmrequest.V1DataQueuesIDPut{
		Name:           name,
		Detail:         detail,
		RoutingMethod:  routingMethod,
		TagIDs:         tagIDs,
		WaitFlowID:     waitFlowID,
		WaitTimeout:    waitTimeout,
		ServiceTimeout: serviceTimeout,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPut, "queue/queues/<queue-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res qmqueue.Queue
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// QueueV1QueueUpdate sends the request to update the queue's tag_ids.
func (r *requestHandler) QueueV1QueueUpdateTagIDs(ctx context.Context, queueID uuid.UUID, tagIDs []uuid.UUID) (*qmqueue.Queue, error) {
	uri := fmt.Sprintf("/v1/queues/%s/tag_ids", queueID)

	data := &qmrequest.V1DataQueuesIDTagIDsPut{
		TagIDs: tagIDs,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPut, "queue/queues/<queue-id>/tag_ids", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res qmqueue.Queue
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// QueueV1QueueGetAgents sends the request to getting the agent list of the given queue.
func (r *requestHandler) QueueV1QueueGetAgents(ctx context.Context, queueID uuid.UUID, filters map[amagent.Field]any) ([]amagent.Agent, error) {
	uri := fmt.Sprintf("/v1/queues/%s/agents", queueID)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodGet, "queue/queues/<queue-id>/agents", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []amagent.Agent
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// QueueV1QueueUpdateRoutingMethod sends the request to update the queue's routing_method.
func (r *requestHandler) QueueV1QueueUpdateRoutingMethod(ctx context.Context, queueID uuid.UUID, routingMethod qmqueue.RoutingMethod) (*qmqueue.Queue, error) {
	uri := fmt.Sprintf("/v1/queues/%s/routing_method", queueID)

	data := &qmrequest.V1DataQueuesIDRoutingMethodPut{
		RoutingMethod: string(routingMethod),
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPut, "queue/queues/<queue-id>/routing_method", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res qmqueue.Queue
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// QueueV1QueueCreateQueuecall sends the request to create a new queuecall.
func (r *requestHandler) QueueV1QueueCreateQueuecall(
	ctx context.Context,
	queueID uuid.UUID,
	referenceType qmqueuecall.ReferenceType,
	referenceID uuid.UUID,
	referenceActiveflowID uuid.UUID,
	exitActionID uuid.UUID,
) (*qmqueuecall.Queuecall, error) {
	uri := fmt.Sprintf("/v1/queues/%s/queuecalls", queueID)

	data := &qmrequest.V1DataQueuesIDQueuecallsPost{
		ReferenceType:         string(referenceType),
		ReferenceID:           referenceID,
		ReferenceActiveflowID: referenceActiveflowID,
		ExitActionID:          exitActionID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPost, "queue/queues/<queue-id>/queuecalls", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res qmqueuecall.Queuecall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// QueueV1QueueExecute sends the request to execute the queue.
// executeDelay: ms
func (r *requestHandler) QueueV1QueueExecuteRun(ctx context.Context, queueID uuid.UUID, executeDelay int) error {
	uri := fmt.Sprintf("/v1/queues/%s/execute_run", queueID)

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPost, "queue/queues/<queue-id>/execute_run", requestTimeoutDefault, executeDelay, ContentTypeJSON, nil)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}

// QueueV1QueueExecute sends the request to execute the queue.
func (r *requestHandler) QueueV1QueueUpdateExecute(ctx context.Context, queueID uuid.UUID, execute qmqueue.Execute) (*qmqueue.Queue, error) {
	uri := fmt.Sprintf("/v1/queues/%s/execute", queueID)

	data := &qmrequest.V1DataQueuesIDExecutePut{
		Execute: execute,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPut, "queue/queues/<queue-id>/execute", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res qmqueue.Queue
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
