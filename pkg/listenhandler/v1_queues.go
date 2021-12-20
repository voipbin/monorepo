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

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/listenhandler/models/request"
)

// processV1QueuesPost handles Post /v1/queues request
func (h *listenHandler) processV1QueuesPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "processV1QueuesPost",
		})
	log.Debug("Executing processV1QueuesPost.")

	var req request.V1DataQueuesPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log = log.WithFields(logrus.Fields{
		"user_id": req.UserID,
		"name":    req.Name,
		"detail":  req.Detail,
	})
	log.Debug("Creating a new queue.")

	// create a new queue
	tmp, err := h.queueHandler.Create(
		ctx,
		req.UserID,
		req.Name,
		req.Detail,
		req.WebhookURI,
		req.WebhookMethod,
		queue.RoutingMethod(req.RoutingMethod),
		req.TagIDs,
		req.WaitActions,
		req.WaitTimout,
		req.ServiceTimeout,
	)
	if err != nil {
		log.Errorf("Could not create the queue. err: %v", err)
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

// processV1QueuesGet handles Get /v1/queues request
func (h *listenHandler) processV1QueuesGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get user_id
	tmpUserID, _ := strconv.Atoi(u.Query().Get("user_id"))
	userID := uint64(tmpUserID)

	log := logrus.WithFields(logrus.Fields{
		"func":  "processV1QueuesGet",
		"user":  userID,
		"size":  pageSize,
		"token": pageToken,
	})

	tmp, err := h.queueHandler.Gets(ctx, userID, pageSize, pageToken)
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

// processV1QueuesIDGet handles Get /v1/queues/<queue-id> request
func (h *listenHandler) processV1QueuesIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "processV1QueuesIDGet",
			"queue_id": id,
		})
	log.Debug("Executing processV1QueuesIDGet.")

	// get queue
	tmp, err := h.queueHandler.Get(ctx, id)
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

// processV1QueuesIDPut handles Put /v1/queues/<queue-id> request
func (h *listenHandler) processV1QueuesIDPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "processV1QueuesIDPut",
			"queue_id": id,
		})
	log.Debug("Executing processV1QueuesIDPut.")

	var req request.V1DataQueuesIDPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log.WithField("request", req).Debug("Updating the queue's basic info.")

	// join to the queue
	err := h.queueHandler.UpdateBasicInfo(ctx, id, req.Name, req.Detail, req.WebhookURI, req.WebhookMethod)
	if err != nil {
		log.Errorf("Could not update the queue's basic info. err: %v", err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// processV1QueuesIDQueuecallsPost handles Post /v1/queues/<queue-id>/queuecalls request
func (h *listenHandler) processV1QueuesIDQueuecallsPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "processV1QueuesIDQueuecallsPost",
			"queue_id": id,
		})
	log.Debug("Executing processV1QueuesIDQueuecallsPost.")

	var req request.V1DataQueuesIDQueuecallsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log = log.WithFields(logrus.Fields{
		"reference_type": req.ReferenceType,
		"reference_id":   req.ReferenceID,
	})
	log.Debug("Joining to the queue.")

	// join to the queue
	tmp, err := h.queueHandler.Join(ctx, id, queuecall.ReferenceType(req.ReferenceType), req.ReferenceID, req.ExitActionID)
	if err != nil {
		log.Errorf("Could not joining to the queue. err: %v", err)
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

// processV1QueuesIDTagIDsPut handles Put /v1/queues/<queue-id>/tag_ids request
func (h *listenHandler) processV1QueuesIDTagIDsPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "processV1QueuesIDTagIDsPut",
			"queue_id": id,
		})
	log.Debug("Executing processV1QueuesIDTagIDsPut.")

	var req request.V1DataQueuesIDTagIDsPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log.WithField("request", req).Debug("Updating the queue's tag_ids info.")

	// update the queue
	err := h.queueHandler.UpdateTagIDs(ctx, id, req.TagIDs)
	if err != nil {
		log.Errorf("Could not update the queue's tag_ids info. err: %v", err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// processV1QueuesIDRoutingMethodPut handles Put /v1/queues/<queue-id>/routing_method request
func (h *listenHandler) processV1QueuesIDRoutingMethodPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "processV1QueuesIDRoutingMethodPut",
			"queue_id": id,
		})
	log.Debug("Executing processV1QueuesIDRoutingMethodPut.")

	var req request.V1DataQueuesIDRoutingMethodPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log.WithField("request", req).Debug("Updating the queue's tag_ids info.")

	// update the queue
	err := h.queueHandler.UpdateRoutingMethod(ctx, id, queue.RoutingMethod(req.RoutingMethod))
	if err != nil {
		log.Errorf("Could not update the queue's routing_method info. err: %v", err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// processV1QueuesIDWaitActionsPut handles Put /v1/queues/<queue-id>/routing_method request
func (h *listenHandler) processV1QueuesIDWaitActionsPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "processV1QueuesIDWaitActionsPut",
			"queue_id": id,
		})
	log.Debug("Executing processV1QueuesIDWaitActionsPut.")

	var req request.V1DataQueuesIDWaitActionsPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log.WithField("request", req).Debug("Updating the queue's tag_ids info.")

	// update the queue
	err := h.queueHandler.UpdateWaitActionsAndTimeouts(ctx, id, req.WaitActions, req.WaitTimeout, req.ServiceTimeout)
	if err != nil {
		log.Errorf("Could not update the queue's wait_actions info. err: %v", err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}
