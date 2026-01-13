package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"
	qmrequest "monorepo/bin-queue-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// QueueV1QueuecallGets sends a request to queue-manager
// to get a list of queuecalls.
// Returns list of queuecalls
func (r *requestHandler) QueueV1QueuecallGets(ctx context.Context, pageToken string, pageSize uint64, filters map[qmqueuecall.Field]any) ([]qmqueuecall.Queuecall, error) {
	uri := fmt.Sprintf("/v1/queuecalls?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodGet, "queue/queuecalls", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []qmqueuecall.Queuecall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// QueueV1QueuecallGet sends a request to queue-manager
// to get the queuecall.
// it returns an queuecall if it succeed.
func (r *requestHandler) QueueV1QueuecallGet(ctx context.Context, queuecallID uuid.UUID) (*qmqueuecall.Queuecall, error) {
	uri := fmt.Sprintf("/v1/queuecalls/%s", queuecallID)

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodGet, "queue/queuecalls/<queuecall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res qmqueuecall.Queuecall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// QueueV1QueuecallGetByReferenceID sends a request to queue-manager
// to get the queuecall of the given reference id.
// it returns an queuecall if it succeed.
func (r *requestHandler) QueueV1QueuecallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*qmqueuecall.Queuecall, error) {
	uri := fmt.Sprintf("/v1/queuecalls/reference_id/%s", referenceID)

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodGet, "queue/queuecalls/<queuecall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res qmqueuecall.Queuecall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// QueueV1QueuecallDelete sends a request to queue-manager
// to delete the queuecall.
func (r *requestHandler) QueueV1QueuecallDelete(ctx context.Context, queuecallID uuid.UUID) (*qmqueuecall.Queuecall, error) {
	uri := fmt.Sprintf("/v1/queuecalls/%s", queuecallID)

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodDelete, "queue/queuecalls/<queuecall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res qmqueuecall.Queuecall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// QueueV1QueuecallKick sends a request to queue-manager
// to kick the queuecall.
func (r *requestHandler) QueueV1QueuecallKick(ctx context.Context, queuecallID uuid.UUID) (*qmqueuecall.Queuecall, error) {
	uri := fmt.Sprintf("/v1/queuecalls/%s/kick", queuecallID)

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPost, "queue/queuecalls/<queuecall-id>/kick", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res qmqueuecall.Queuecall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// QueueV1QueuecallKick sends a request to queue-manager
// to kick the queuecall.
func (r *requestHandler) QueueV1QueuecallKickByReferenceID(ctx context.Context, referenceID uuid.UUID) (*qmqueuecall.Queuecall, error) {
	uri := fmt.Sprintf("/v1/queuecalls/reference_id/%s/kick", referenceID)

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPost, "queue/queuecalls/reference_id/<reference-id>/kick", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res qmqueuecall.Queuecall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// QueueV1QueuecallTimeoutWait sends the request for queuecall wait timeout.
//
// delay: millisecond
func (r *requestHandler) QueueV1QueuecallTimeoutWait(ctx context.Context, queuecallID uuid.UUID, delay int) error {
	uri := fmt.Sprintf("/v1/queuecalls/%s/timeout_wait", queuecallID)

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPost, "queue/queuecalls/<queuecall-id>/timeout_wait", requestTimeoutDefault, delay, ContentTypeNone, nil)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}

// QueueV1QueuecallTimeoutService sends the request for queuecall service timeout.
//
// delay: millisecond
func (r *requestHandler) QueueV1QueuecallTimeoutService(ctx context.Context, queuecallID uuid.UUID, delay int) error {
	uri := fmt.Sprintf("/v1/queuecalls/%s/timeout_service", queuecallID)

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPost, "queue/queuecalls/<queuecall-id>/timeout_service", requestTimeoutDefault, delay, ContentTypeNone, nil)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}

// QueueV1QueuecallUpdateStatusWaiting sends the request for update the queuecall status to waiting.
func (r *requestHandler) QueueV1QueuecallUpdateStatusWaiting(ctx context.Context, queuecallID uuid.UUID) (*qmqueuecall.Queuecall, error) {
	uri := fmt.Sprintf("/v1/queuecalls/%s/status_waiting", queuecallID)

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPost, "queue/queuecalls", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res qmqueuecall.Queuecall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// QueueV1QueuecallExecute sends the request for queuecall execute.
func (r *requestHandler) QueueV1QueuecallExecute(ctx context.Context, queuecallID uuid.UUID, agentID uuid.UUID) (*qmqueuecall.Queuecall, error) {
	uri := fmt.Sprintf("/v1/queuecalls/%s/execute", queuecallID)

	data := &qmrequest.V1DataQueuecallsIDExecutePost{
		AgentID: agentID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPost, "queue/queuecalls", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res qmqueuecall.Queuecall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// QueueV1QueuecallHealthCheck sends the request for queuecall health-check
//
// delay: milliseconds
func (r *requestHandler) QueueV1QueuecallHealthCheck(ctx context.Context, id uuid.UUID, delay int, retryCount int) error {
	uri := fmt.Sprintf("/v1/queuecalls/%s/health-check", id)

	m, err := json.Marshal(qmrequest.V1DataQueuecallsIDHealthCheckPost{
		RetryCount: retryCount,
	})
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestQueue(ctx, uri, sock.RequestMethodPost, "queue/queuecalls/<queuecall-id>/health-check", requestTimeoutDefault, delay, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}
