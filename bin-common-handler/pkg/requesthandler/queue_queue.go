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
)

// QueueV1QueueGets sends a request to queue-manager
// to get a list of queues.
// Returns list of queues
func (r *requestHandler) QueueV1QueueGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]qmqueue.Queue, error) {
	uri := fmt.Sprintf("/v1/queues?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	res, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodGet, "queue/queues", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData []qmqueue.Queue
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return resData, nil
}

// QueueV1QueueGet sends a request to queue-manager
// to getting the queue.
// it returns an queue if it succeed.
func (r *requestHandler) QueueV1QueueGet(ctx context.Context, queueID uuid.UUID) (*qmqueue.Queue, error) {
	uri := fmt.Sprintf("/v1/queues/%s", queueID)

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodGet, "queue/queues", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res qmqueue.Queue
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
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

	res, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPost, "queue/queues", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var c qmqueue.Queue
	if err := json.Unmarshal([]byte(res.Data), &c); err != nil {
		return nil, err
	}

	return &c, nil
}

// QueueV1QueueDelete sends a request to queue-manager
// to deleteing the queue.
// it returns an error if it failed.
func (r *requestHandler) QueueV1QueueDelete(ctx context.Context, queueID uuid.UUID) (*qmqueue.Queue, error) {
	uri := fmt.Sprintf("/v1/queues/%s", queueID)

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodDelete, "queue/queues", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res qmqueue.Queue
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
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

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPut, "queue/queues", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res qmqueue.Queue
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
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

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPut, "queue/queues", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res qmqueue.Queue
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// QueueV1QueueGetAgents sends the request to getting the agent list of the given queue's and status.
func (r *requestHandler) QueueV1QueueGetAgents(ctx context.Context, queueID uuid.UUID, status amagent.Status) ([]amagent.Agent, error) {
	uri := fmt.Sprintf("/v1/queues/%s/agents?status=%s", queueID, status)

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodGet, "queue/queues", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []amagent.Agent
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
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

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPut, "queue/queues", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res qmqueue.Queue
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
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

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPost, "queue/queues", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var c qmqueuecall.Queuecall
	if err := json.Unmarshal([]byte(tmp.Data), &c); err != nil {
		return nil, err
	}

	return &c, nil
}

// QueueV1QueueExecute sends the request to execute the queue.
// executeDelay: ms
func (r *requestHandler) QueueV1QueueExecuteRun(ctx context.Context, queueID uuid.UUID, executeDelay int) error {
	uri := fmt.Sprintf("/v1/queues/%s/execute_run", queueID)

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPost, "queue/queues", requestTimeoutDefault, executeDelay, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		return nil
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
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

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPut, "queue/queues", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var c qmqueue.Queue
	if err := json.Unmarshal([]byte(tmp.Data), &c); err != nil {
		return nil, err
	}

	return &c, nil
}
