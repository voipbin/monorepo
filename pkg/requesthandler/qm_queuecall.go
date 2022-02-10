package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	qmqueuecall "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	qmrequest "gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// QMV1QueuecallGets sends a request to queue-manager
// to get a list of queuecalls.
// Returns list of queuecalls
func (r *requestHandler) QMV1QueuecallGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]qmqueuecall.Queuecall, error) {
	uri := fmt.Sprintf("/v1/queuecalls?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	res, err := r.sendRequestQM(uri, rabbitmqhandler.RequestMethodGet, resourceQMQueuecalls, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData []qmqueuecall.Queuecall
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return resData, nil
}

// QMV1QueuecallGet sends a request to queue-manager
// to get the queuecall.
// it returns an queuecall if it succeed.
func (r *requestHandler) QMV1QueuecallGet(ctx context.Context, queuecallID uuid.UUID) (*qmqueuecall.Queuecall, error) {
	uri := fmt.Sprintf("/v1/queuecalls/%s", queuecallID)

	tmp, err := r.sendRequestQM(uri, rabbitmqhandler.RequestMethodGet, resourceQMQueuecalls, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// QMV1QueuecallDelete sends a request to queue-manager
// to delete the queuecall and exit the queuecall from the queue.
func (r *requestHandler) QMV1QueuecallDelete(ctx context.Context, queuecallID uuid.UUID) (*qmqueuecall.Queuecall, error) {
	uri := fmt.Sprintf("/v1/queuecalls/%s", queuecallID)

	tmp, err := r.sendRequestQM(uri, rabbitmqhandler.RequestMethodDelete, resourceQMQueuecalls, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// QMV1QueuecallreferenceDelete sends a request to queue-manager
// to delete the queuecallreference and exit the queuecall from the queue.
func (r *requestHandler) QMV1QueuecallDeleteByReferenceID(ctx context.Context, referenceID uuid.UUID) (*qmqueuecall.Queuecall, error) {
	uri := fmt.Sprintf("/v1/queuecallreferences/%s", referenceID)

	tmp, err := r.sendRequestQM(uri, rabbitmqhandler.RequestMethodDelete, resourceQMQueuecallreferences, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// QMV1QueuecallExecute sends the request for queuecall execution.
//
// delay: millisecond
func (r *requestHandler) QMV1QueuecallExecute(ctx context.Context, queuecallID uuid.UUID, searchDelay int) (*qmqueuecall.Queuecall, error) {
	uri := fmt.Sprintf("/v1/queuecalls/%s/execute", queuecallID)

	data := &qmrequest.V1DataQueuesIDExecutePost{
		SearchDelay: searchDelay,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestQM(uri, rabbitmqhandler.RequestMethodPost, resourceQMQueuecalls, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// QMV1QueuecallSearchAgent sends the request for queuecall agent search and make agentcall.
//
// delay: millisecond
func (r *requestHandler) QMV1QueuecallSearchAgent(ctx context.Context, queuecallID uuid.UUID, delay int) error {
	uri := fmt.Sprintf("/v1/queuecalls/%s/search_agent", queuecallID)

	res, err := r.sendRequestQM(uri, rabbitmqhandler.RequestMethodPost, resourceQMQueuecalls, requestTimeoutDefault, delay, ContentTypeJSON, nil)
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

// QMV1QueuecallTimeoutWait sends the request for queuecall wait timeout.
//
// delay: millisecond
func (r *requestHandler) QMV1QueuecallTimeoutWait(ctx context.Context, queuecallID uuid.UUID, delay int) error {
	uri := fmt.Sprintf("/v1/queuecalls/%s/timeout_wait", queuecallID)

	res, err := r.sendRequestQM(uri, rabbitmqhandler.RequestMethodPost, resourceQMQueuecalls, requestTimeoutDefault, delay, ContentTypeJSON, nil)
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

// QMV1QueuecallTimeoutService sends the request for queuecall service timeout.
//
// delay: millisecond
func (r *requestHandler) QMV1QueuecallTimeoutService(ctx context.Context, queuecallID uuid.UUID, delay int) error {
	uri := fmt.Sprintf("/v1/queuecalls/%s/timeout_service", queuecallID)

	res, err := r.sendRequestQM(uri, rabbitmqhandler.RequestMethodPost, resourceQMQueuecalls, requestTimeoutDefault, delay, ContentTypeJSON, nil)
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
