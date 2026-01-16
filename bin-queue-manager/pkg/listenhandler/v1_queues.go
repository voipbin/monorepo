package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/pkg/listenhandler/models/request"
)

// processV1QueuesPost handles Post /v1/queues request
func (h *listenHandler) processV1QueuesPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1QueuesPost",
		"request": m,
	})

	var req request.V1DataQueuesPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	// create a new queue
	tmp, err := h.queueHandler.Create(
		ctx,
		req.CustomerID,
		req.Name,
		req.Detail,
		queue.RoutingMethod(req.RoutingMethod),
		req.TagIDs,
		req.WaitFlowID,
		req.WaitTimeout,
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

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1QueuesGet handles Get /v1/queues request
func (h *listenHandler) processV1QueuesGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1QueuesGet",
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
	tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(m.Data)
	if err != nil {
		log.Errorf("Could not parse filters. err: %v", err)
		return simpleResponse(400), nil
	}

	// convert to typed filters
	filters, err := utilhandler.ConvertFilters[queue.FieldStruct, queue.Field](queue.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.queueHandler.List(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get queue info. err: %v", err)
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

// processV1QueuesIDGet handles Get /v1/queues/<queue-id> request
func (h *listenHandler) processV1QueuesIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1QueuesIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

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

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1QueuesIDDelete handles Delete /v1/queues/<queue-id> request
func (h *listenHandler) processV1QueuesIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1QueuesIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.queueHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete queue info. err: %v", err)
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

// processV1QueuesIDPut handles Put /v1/queues/<queue-id> request
func (h *listenHandler) processV1QueuesIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1QueuesIDPut",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataQueuesIDPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	// join to the queue
	tmp, err := h.queueHandler.UpdateBasicInfo(
		ctx,
		id,
		req.Name,
		req.Detail,
		req.RoutingMethod,
		req.TagIDs,
		req.WaitFlowID,
		req.WaitTimeout,
		req.ServiceTimeout,
	)
	if err != nil {
		log.Errorf("Could not update the queue info. err: %v", err)
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

// processV1QueuesIDTagIDsPut handles Put /v1/queues/<queue-id>/tag_ids request
func (h *listenHandler) processV1QueuesIDTagIDsPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1QueuesIDTagIDsPut",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataQueuesIDTagIDsPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	// update the queue
	tmp, err := h.queueHandler.UpdateTagIDs(ctx, id, req.TagIDs)
	if err != nil {
		log.Errorf("Could not get queue info. err: %v", err)
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

// processV1QueuesIDRoutingMethodPut handles Put /v1/queues/<queue-id>/routing_method request
func (h *listenHandler) processV1QueuesIDRoutingMethodPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1QueuesIDRoutingMethodPut",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataQueuesIDRoutingMethodPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	// update the queue
	tmp, err := h.queueHandler.UpdateRoutingMethod(ctx, id, queue.RoutingMethod(req.RoutingMethod))
	if err != nil {
		log.Errorf("Could not update the queue info. err: %v", err)
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

// // processV1QueuesIDWaitActionsPut handles Put /v1/queues/<queue-id>/wait_actions request
// func (h *listenHandler) processV1QueuesIDWaitActionsPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func":    "processV1QueuesIDWaitActionsPut",
// 		"request": m,
// 	})

// 	uriItems := strings.Split(m.URI, "/")
// 	if len(uriItems) < 5 {
// 		return simpleResponse(400), nil
// 	}

// 	id := uuid.FromStringOrNil(uriItems[3])

// 	var req request.V1DataQueuesIDWaitActionsPut
// 	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
// 		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
// 		return simpleResponse(400), nil
// 	}

// 	// update the queue
// 	tmp, err := h.queueHandler.UpdateWaitActionsAndTimeouts(ctx, id, req.WaitActions, req.WaitTimeout, req.ServiceTimeout)
// 	if err != nil {
// 		log.Errorf("Could not update the queue info. err: %v", err)
// 		return simpleResponse(500), nil
// 	}

// 	data, err := json.Marshal(tmp)
// 	if err != nil {
// 		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
// 		return simpleResponse(500), nil
// 	}

// 	res := &sock.Response{
// 		StatusCode: 200,
// 		DataType:   "application/json",
// 		Data:       data,
// 	}

// 	return res, nil
// }

// processV1QueuesIDAgentsGet handles Get /v1/queues/<queue-id>/agents request
func (h *listenHandler) processV1QueuesIDAgentsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1QueuesIDAgentsGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	// Parse filters from request body
	var req request.V1DataQueuesIDAgentsGet
	if len(m.Data) > 0 {
		if err := json.Unmarshal(m.Data, &req); err != nil {
			log.Errorf("Could not unmarshal filters. err: %v", err)
			return simpleResponse(400), nil
		}
	}

	log.WithFields(logrus.Fields{
		"status":           req.Status,
		"filters_raw_data": string(m.Data),
	}).Debug("processV1QueuesIDAgentsGet: Parsed filters from request body")

	status := amagent.Status(req.Status)
	tmp, err := h.queueHandler.GetAgents(ctx, id, status)
	if err != nil {
		log.Errorf("Could not get agents. err: %v", err)
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

// processV1QueuesIDExecuteRunPost handles Post /v1/queues/<queue-id>/execute_run request
func (h *listenHandler) processV1QueuesIDExecuteRunPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1QueuesIDExecuteRunPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		log.Errorf("Wrong uri.")
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	h.queueHandler.Execute(ctx, id)

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// processV1QueuesIDExecutePut handles Put /v1/queues/<queue-id>/execute request
func (h *listenHandler) processV1QueuesIDExecutePut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1QueuesIDExecutePut",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataQueuesIDExecutePut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.queueHandler.UpdateExecute(ctx, id, req.Execute)
	if err != nil {
		log.Errorf("Could not get agents. err: %v", err)
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

// processV1QueuecallsIDStatusWaitingPost handles Post /v1/queuecalls/<queuecall-id>/status_waiting request
func (h *listenHandler) processV1QueuecallsIDStatusWaitingPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1QueuecallsIDStatusWaitingPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.queuecallHandler.UpdateStatusWaiting(ctx, id)
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
