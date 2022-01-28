package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processV1QueuecallsGet handles Get /v1/queuecalls request
func (h *listenHandler) processV1QueuecallsGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get user_id
	customerID := uuid.FromStringOrNil(u.Query().Get("customer_id"))

	log := logrus.WithFields(logrus.Fields{
		"func":        "processV1QueuecallsGet",
		"customer_id": customerID,
		"size":        pageSize,
		"token":       pageToken,
	})

	tmp, err := h.queuecallHandler.Gets(ctx, customerID, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get queuecalls info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1QueuecallsIDGet handles Get /v1/queuecalls/<queue-id> request
func (h *listenHandler) processV1QueuecallsIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "processV1QueuecallsIDGet",
			"queue_id": id,
		})
	log.Debug("Executing processV1QueuecallsIDGet.")

	// get queue
	tmp, err := h.queuecallHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get queue info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1QueuecallsIDDelete handles Delete /v1/queuecalls/<queuecall-id> request
func (h *listenHandler) processV1QueuecallsIDDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "processV1QueuecallsIDDelete",
			"queuecall_id": id,
		})
	log.Debug("Executing processV1QueuecallsIDDelete.")

	if err := h.queuecallHandler.Kick(ctx, id); err != nil {
		log.Errorf("Could not leave the queuecall from the queue. err: %v", err)
		return simpleResponse(500), nil
	}
	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// processV1QueuecallsIDExecutePost handles Post /v1/queuecalls/<queuecall-id>/execute request
func (h *listenHandler) processV1QueuecallsIDExecutePost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "processV1QueuecallsIDExecutePost",
			"queuecall_id": id,
		})
	log.Debug("Executing processV1QueuecallsIDExecutePost.")

	h.queuecallHandler.Execute(ctx, id)
	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// processV1QueuecallsIDTimeoutWaitPost handles Post /v1/queuecalls/<queuecall-id>/timeout_wait request
func (h *listenHandler) processV1QueuecallsIDTimeoutWaitPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "processV1QueuecallsIDTimeoutWaitPost",
			"queuecall_id": id,
		})
	log.Debug("Executing processV1QueuecallsIDTimeoutWaitPost.")

	h.queuecallHandler.TimeoutWait(ctx, id)
	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// processV1QueuecallsIDTimeoutServicePost handles Post /v1/queuecalls/<queuecall-id>/timeout_service request
func (h *listenHandler) processV1QueuecallsIDTimeoutServicePost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "processV1QueuecallsIDTimeoutServicePost",
			"queuecall_id": id,
		})
	log.Debug("Executing processV1QueuecallsIDTimeoutServicePost.")

	h.queuecallHandler.TimeoutService(ctx, id)
	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}
