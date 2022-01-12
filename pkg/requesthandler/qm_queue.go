package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	qmqueue "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	qmqueuecall "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	qmrequest "gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/listenhandler/models/request"
)

// QMV1QueueGets sends a request to queue-manager
// to get a list of queues.
// Returns list of queues
func (r *requestHandler) QMV1QueueGets(ctx context.Context, userID uint64, pageToken string, pageSize uint64) ([]qmqueue.Queue, error) {
	uri := fmt.Sprintf("/v1/queues?page_token=%s&page_size=%d&user_id=%d", url.QueryEscape(pageToken), pageSize, userID)

	res, err := r.sendRequestQM(uri, rabbitmqhandler.RequestMethodGet, resourceQMQueues, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// QMV1QueueGet sends a request to queue-manager
// to getting the queue.
// it returns an queue if it succeed.
func (r *requestHandler) QMV1QueueGet(ctx context.Context, queueID uuid.UUID) (*qmqueue.Queue, error) {
	uri := fmt.Sprintf("/v1/queues/%s", queueID)

	tmp, err := r.sendRequestQM(uri, rabbitmqhandler.RequestMethodGet, resourceQMQueues, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// QMV1QueueCreate sends the request to create the queue.
func (r *requestHandler) QMV1QueueCreate(ctx context.Context, userID uint64, name, detail, webhookURI, webhookMethod string, routingMethod qmqueue.RoutingMethod, tagIDs []uuid.UUID, waitActions []fmaction.Action, timeoutWait, timeoutService int) (*qmqueue.Queue, error) {
	uri := "/v1/queues"

	data := &qmrequest.V1DataQueuesPost{
		UserID:         userID,
		Name:           name,
		Detail:         detail,
		WebhookURI:     webhookURI,
		WebhookMethod:  webhookMethod,
		RoutingMethod:  string(routingMethod),
		TagIDs:         tagIDs,
		WaitActions:    waitActions,
		WaitTimout:     timeoutWait,
		ServiceTimeout: timeoutService,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestQM(uri, rabbitmqhandler.RequestMethodPost, resourceQMQueues, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// QMV1QueueDelete sends a request to queue-manager
// to deleteing the queue.
// it returns an error if it failed.
func (r *requestHandler) QMV1QueueDelete(ctx context.Context, queueID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/queues/%s", queueID)

	tmp, err := r.sendRequestQM(uri, rabbitmqhandler.RequestMethodDelete, resourceQMQueues, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

// QMV1QueueUpdate sends the request to update the queue.
func (r *requestHandler) QMV1QueueUpdate(ctx context.Context, queueID uuid.UUID, name, detail, webhookURI, webhookMethod string) error {
	uri := fmt.Sprintf("/v1/queues/%s", queueID)

	data := &qmrequest.V1DataQueuesIDPut{
		Name:          name,
		Detail:        detail,
		WebhookURI:    webhookURI,
		WebhookMethod: webhookMethod,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestQM(uri, rabbitmqhandler.RequestMethodPut, resourceQMQueues, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

// QMV1QueueUpdate sends the request to update the queue's tag_ids.
func (r *requestHandler) QMV1QueueUpdateTagIDs(ctx context.Context, queueID uuid.UUID, tagIDs []uuid.UUID) error {
	uri := fmt.Sprintf("/v1/queues/%s/tag_ids", queueID)

	data := &qmrequest.V1DataQueuesIDTagIDsPut{
		TagIDs: tagIDs,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestQM(uri, rabbitmqhandler.RequestMethodPut, resourceQMQueues, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

// QMV1QueueUpdateRoutingMethod sends the request to update the queue's routing_method.
func (r *requestHandler) QMV1QueueUpdateRoutingMethod(ctx context.Context, queueID uuid.UUID, routingMethod qmqueue.RoutingMethod) error {
	uri := fmt.Sprintf("/v1/queues/%s/routing_method", queueID)

	data := &qmrequest.V1DataQueuesIDRoutingMethodPut{
		RoutingMethod: string(routingMethod),
	}

	m, err := json.Marshal(data)
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestQM(uri, rabbitmqhandler.RequestMethodPut, resourceQMQueues, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

// QMV1QueueUpdateActions sends the request to update the queue's action handles.
func (r *requestHandler) QMV1QueueUpdateActions(ctx context.Context, queueID uuid.UUID, waitActions []fmaction.Action, timeoutWait, timeoutService int) error {
	uri := fmt.Sprintf("/v1/queues/%s/wait_actions", queueID)

	data := &qmrequest.V1DataQueuesIDWaitActionsPut{
		WaitActions:    waitActions,
		WaitTimeout:    timeoutWait,
		ServiceTimeout: timeoutService,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestQM(uri, rabbitmqhandler.RequestMethodPut, resourceQMQueues, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

// QMV1QueueUpdateActions sends the request to update the queue's action handles.
func (r *requestHandler) QMV1QueueCreateQueuecall(ctx context.Context, queueID uuid.UUID, referenceType qmqueuecall.ReferenceType, referenceID, exitActionID uuid.UUID) (*qmqueuecall.Queuecall, error) {
	uri := fmt.Sprintf("/v1/queues/%s/queuecalls", queueID)

	data := &qmrequest.V1DataQueuesIDQueuecallsPost{
		ReferenceType: string(referenceType),
		ReferenceID:   referenceID,
		ExitActionID:  exitActionID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestQM(uri, rabbitmqhandler.RequestMethodPost, resourceQMQueues, requestTimeoutDefault, 0, ContentTypeJSON, m)
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
