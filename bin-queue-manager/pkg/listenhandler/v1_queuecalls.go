package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/listenhandler/models/request"
)

// processV1QueuecallsGet handles Get /v1/queuecalls request
func (h *listenHandler) processV1QueuecallsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1QueuecallsGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get filters from request body
	var req map[string]any
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	filters, err := queuecall.ConvertStringMapToFieldMap(req)
	if err != nil {
		log.Debugf("Could not convert the filters. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.queuecallHandler.Gets(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get queuecalls info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1QueuecallsIDGet handles Get /v1/queuecalls/reference_id/<reference-id> request
func (h *listenHandler) processV1QueuecallsReferenceIDIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1QueuecallsReferenceIDIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	referenceID := uuid.FromStringOrNil(uriItems[4])

	// get queue
	tmp, err := h.queuecallHandler.GetByReferenceID(ctx, referenceID)
	if err != nil {
		log.Errorf("Could not get queuecall info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1QueuecallsIDGet handles Get /v1/queuecalls/<queue-id> request
func (h *listenHandler) processV1QueuecallsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1QueuecallsIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	// get queue
	tmp, err := h.queuecallHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get queuecall info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1QueuecallsIDDelete handles Delete /v1/queuecalls/<queuecall-id> request
func (h *listenHandler) processV1QueuecallsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1QueuecallsIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.queuecallHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not leave the queuecall from the queue. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1QueuecallsIDTimeoutWaitPost handles Post /v1/queuecalls/<queuecall-id>/timeout_wait request
func (h *listenHandler) processV1QueuecallsIDTimeoutWaitPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1QueuecallsIDTimeoutWaitPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		log.Errorf("Wrong uri.")
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	h.queuecallHandler.TimeoutWait(ctx, id)
	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// processV1QueuecallsIDTimeoutServicePost handles Post /v1/queuecalls/<queuecall-id>/timeout_service request
func (h *listenHandler) processV1QueuecallsIDTimeoutServicePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1QueuecallsIDTimeoutServicePost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		log.Errorf("Wrong uri.")
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	h.queuecallHandler.TimeoutService(ctx, id)
	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// processV1QueuecallsIDExecutePost handles Post /v1/queuecalls/<queuecall-id>/execute request
func (h *listenHandler) processV1QueuecallsIDExecutePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1QueuecallsIDExecutePost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataQueuecallsIDExecutePost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.queuecallHandler.Execute(ctx, id, req.AgentID)
	if err != nil {
		log.Errorf("Could not leave the queuecall from the queue. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1QueuecallsIDKickPost handles Post /v1/queuecalls/<queuecall-id>/kick request
func (h *listenHandler) processV1QueuecallsIDKickPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1QueuecallsIDIDKickPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.queuecallHandler.Kick(ctx, id)
	if err != nil {
		log.Errorf("Could not leave the queuecall from the queue. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1QueuecallsIDKickPost handles Post /v1/queuecalls/reference_id/<reference-id>/kick request
func (h *listenHandler) processV1QueuecallsReferenceIDIDKickPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1QueuecallsReferenceIDKickPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	referenceID := uuid.FromStringOrNil(uriItems[4])

	tmp, err := h.queuecallHandler.KickByReferenceID(ctx, referenceID)
	if err != nil {
		log.Errorf("Could not leave the queuecall from the queue. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1QueuecallsIDHealthCheckPost handles Post /v1/queuecalls/<queuecall-id>/health-check request
func (h *listenHandler) processV1QueuecallsIDHealthCheckPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1QueuecallsIDHealthCheckPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataQueuecallsIDHealthCheckPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	go h.queuecallHandler.HealthCheck(ctx, id, req.RetryCount)
	return nil, nil
}
