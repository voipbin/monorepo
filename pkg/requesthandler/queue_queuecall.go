package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	qmqueuecall "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// QueueV1QueuecallGets sends a request to queue-manager
// to get a list of queuecalls.
// Returns list of queuecalls
func (r *requestHandler) QueueV1QueuecallGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]qmqueuecall.Queuecall, error) {
	uri := fmt.Sprintf("/v1/queuecalls?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	tmp, err := r.sendRequestQueue(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceQueueQueuecalls, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []qmqueuecall.Queuecall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// QueueV1QueuecallGetsByQueueIDAndStatus sends a request to queue-manager
// to get a list of queuecalls by the given queue id and status.
// Returns list of queuecalls
func (r *requestHandler) QueueV1QueuecallGetsByQueueIDAndStatus(ctx context.Context, queueID uuid.UUID, status qmqueuecall.Status, pageToken string, pageSize uint64) ([]qmqueuecall.Queuecall, error) {
	uri := fmt.Sprintf("/v1/queuecalls?page_token=%s&page_size=%d&queue_id=%s&status=%s", url.QueryEscape(pageToken), pageSize, queueID, status)

	tmp, err := r.sendRequestQueue(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceQueueQueuecalls, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []qmqueuecall.Queuecall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// QueueV1QueuecallGet sends a request to queue-manager
// to get the queuecall.
// it returns an queuecall if it succeed.
func (r *requestHandler) QueueV1QueuecallGet(ctx context.Context, queuecallID uuid.UUID) (*qmqueuecall.Queuecall, error) {
	uri := fmt.Sprintf("/v1/queuecalls/%s", queuecallID)

	tmp, err := r.sendRequestQueue(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceQueueQueuecalls, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res qmqueuecall.Queuecall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// QueueV1QueuecallDelete sends a request to queue-manager
// to delete the queuecall and exit the queuecall from the queue.
func (r *requestHandler) QueueV1QueuecallDelete(ctx context.Context, queuecallID uuid.UUID) (*qmqueuecall.Queuecall, error) {
	uri := fmt.Sprintf("/v1/queuecalls/%s", queuecallID)

	tmp, err := r.sendRequestQueue(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceQueueQueuecalls, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res qmqueuecall.Queuecall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// QueueV1QueuecallreferenceDelete sends a request to queue-manager
// to delete the queuecallreference and exit the queuecall from the queue.
func (r *requestHandler) QueueV1QueuecallDeleteByReferenceID(ctx context.Context, referenceID uuid.UUID) (*qmqueuecall.Queuecall, error) {
	uri := fmt.Sprintf("/v1/queuecallreferences/%s", referenceID)

	tmp, err := r.sendRequestQueue(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceQueueQueuecallreferences, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res qmqueuecall.Queuecall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// QueueV1QueuecallTimeoutWait sends the request for queuecall wait timeout.
//
// delay: millisecond
func (r *requestHandler) QueueV1QueuecallTimeoutWait(ctx context.Context, queuecallID uuid.UUID, delay int) error {
	uri := fmt.Sprintf("/v1/queuecalls/%s/timeout_wait", queuecallID)

	res, err := r.sendRequestQueue(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceQueueQueuecalls, requestTimeoutDefault, delay, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case res == nil:
		return nil
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// QueueV1QueuecallTimeoutService sends the request for queuecall service timeout.
//
// delay: millisecond
func (r *requestHandler) QueueV1QueuecallTimeoutService(ctx context.Context, queuecallID uuid.UUID, delay int) error {
	uri := fmt.Sprintf("/v1/queuecalls/%s/timeout_service", queuecallID)

	res, err := r.sendRequestQueue(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceQueueQueuecalls, requestTimeoutDefault, delay, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case res == nil:
		return nil
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// QueueV1QueuecallUpdateStatusWaiting sends the request for update the queuecall status to waiting.
func (r *requestHandler) QueueV1QueuecallUpdateStatusWaiting(ctx context.Context, queuecallID uuid.UUID) (*qmqueuecall.Queuecall, error) {
	uri := fmt.Sprintf("/v1/queuecalls/%s/status_waiting", queuecallID)

	tmp, err := r.sendRequestQueue(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceQueueQueuecalls, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res qmqueuecall.Queuecall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
