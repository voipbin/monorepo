package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/pkg/listenhandler/models/request"
)

// v1ActiveflowsGet handles /v1/activeflows GET request
func (h *listenHandler) v1ActiveflowsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1ActiveflowsGet",
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

	var req map[string]any
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	filters, err := activeflow.ConvertStringMapToFieldMap(req)
	if err != nil {
		log.Errorf("Could not convert the filters. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.activeflowHandler.List(ctx, pageToken, pageSize, filters)
	if err != nil {
		log.Errorf("Could not get activeflows. err: %v", err)
		return nil, errors.Wrap(err, "could not get activeflows")
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ActiveflowsPost handles /v1/activeflows POST request
// creates a new activeflow with given data.
func (h *listenHandler) v1ActiveflowsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1ActiveflowsPost",
		"request": m,
	})

	var reqData request.V1DataActiveFlowsPost
	if err := json.Unmarshal(m.Data, &reqData); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// create active flow
	resActiveFlow, err := h.activeflowHandler.Create(ctx, reqData.ID, reqData.CustomerID, reqData.ReferenceType, reqData.ReferenceID, reqData.ReferenceActiveflowID, reqData.FlowID)
	if err != nil {
		log.Errorf("Could not create a new active flow. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(resActiveFlow)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ActiveflowsIDGet handles
// /v1/activeflows/<activeflow-id> GET
func (h *listenHandler) v1ActiveflowsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1ActiveflowsIDGet",
		"request": m,
	})

	// "/v1/activeflows/be2692f8-066a-11eb-847f-1b4de696fafb"
	tmpVals := strings.Split(m.URI, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.activeflowHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get activeflow. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ActiveflowsIDDelete handles
// /v1/activeflows/{id} DELETE
func (h *listenHandler) v1ActiveflowsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1ActiveflowsIDDelete",
		"request": m,
	})

	// "/v1/activeflows/be2692f8-066a-11eb-847f-1b4de696fafb"
	tmpVals := strings.Split(m.URI, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.activeflowHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete activeflow. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ActiveflowsIDNextGet handles
// /v1/activeflows/{id}/next GET
func (h *listenHandler) v1ActiveflowsIDNextGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1ActiveflowsIDNextGet",
		"request": m,
	})

	// "/v1/activeflows/be2692f8-066a-11eb-847f-1b4de696fafb/next"
	tmpVals := strings.Split(m.URI, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	var req request.V1DataActiveFlowsIDNextGet
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	resAction, err := h.activeflowHandler.ExecuteNextAction(ctx, id, req.CurrentActionID)
	if err != nil {
		log.Errorf("Could not execute the next action. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(resAction)
	if err != nil {
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ActiveflowsIDForwardActionIDPut handles
// /v1/activeflows/{id}/forward_action_id PUT
func (h *listenHandler) v1ActiveflowsIDForwardActionIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1ActiveflowsIDForwardActionIDPut",
		"request": m,
	})

	// "/v1/activeflows/be2692f8-066a-11eb-847f-1b4de696fafb/forward_action_id"
	tmpVals := strings.Split(m.URI, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	var req request.V1DataActiveFlowsIDForwardActionIDPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	if err := h.activeflowHandler.SetForwardActionID(ctx, id, req.ForwardActionID, req.ForwardNow); err != nil {
		log.Errorf("Could not set the forward action id. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// v1ActiveflowsIDExecutePost handles
// /v1/activeflows/{id}/execute Post
func (h *listenHandler) v1ActiveflowsIDExecutePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1ActiveflowsIDExecutePost",
		"request": m,
	})

	// "/v1/activeflows/be2692f8-066a-11eb-847f-1b4de696fafb/execute"
	tmpVals := strings.Split(m.URI, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	go func() {
		if err := h.activeflowHandler.Execute(context.Background(), id); err != nil {
			log.Errorf("Could not execute the activeflow correctly. err: %v", err)
		}
	}()

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// v1ActiveflowsIDStopPost handles
// /v1/activeflows/<activeflow-id>/stop Post
func (h *listenHandler) v1ActiveflowsIDStopPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1ActiveflowsIDStopPost",
		"request": m,
	})

	// "/v1/activeflows/be2692f8-066a-11eb-847f-1b4de696fafb/stop"
	tmpVals := strings.Split(m.URI, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.activeflowHandler.Stop(ctx, id)
	if err != nil {
		log.Errorf("Could not stop the activeflow correctly. err: %v", err)
		return nil, errors.Wrap(err, "Could not stop the activeflow.")
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the result. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ActiveflowsIDPushActionsPost handles
// /v1/activeflows/<activeflow-id>/push_actions Post
func (h *listenHandler) v1ActiveflowsIDPushActionsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1ActiveflowsIDPushActionsPost",
		"request": m,
	})

	// "/v1/activeflows/be2692f8-066a-11eb-847f-1b4de696fafb/push_actions"
	tmpVals := strings.Split(m.URI, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	var req request.V1DataActiveFlowsIDPushActionPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	tmp, err := h.activeflowHandler.PushActions(ctx, id, req.Actions)
	if err != nil {
		log.Errorf("Could not stop the activeflow correctly. err: %v", err)
		return nil, errors.Wrap(err, "Could not stop the activeflow.")
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the result. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ActiveflowsIDAddActionsPost handles
// /v1/activeflows/<activeflow-id>/add_actions Post
func (h *listenHandler) v1ActiveflowsIDAddActionsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1ActiveflowsIDAddActionsPost",
		"request": m,
	})

	// "/v1/activeflows/be2692f8-066a-11eb-847f-1b4de696fafb/add_actions"
	tmpVals := strings.Split(m.URI, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	var req request.V1DataActiveFlowsIDAddActionPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	tmp, err := h.activeflowHandler.AddActions(ctx, id, req.Actions)
	if err != nil {
		log.Errorf("Could not stop the activeflow correctly. err: %v", err)
		return nil, errors.Wrap(err, "Could not stop the activeflow.")
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the result. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ActiveflowsIDServiceStopPost handles
// /v1/activeflows/<activeflow-id>/service_stop Post
func (h *listenHandler) v1ActiveflowsIDServiceStopPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1ActiveflowsIDServiceStopPost",
		"request": m,
	})

	// "/v1/activeflows/be2692f8-066a-11eb-847f-1b4de696fafb/service_stop"
	tmpVals := strings.Split(m.URI, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	var req request.V1DataActiveFlowsIDServiceStopPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	if errStop := h.activeflowHandler.ServiceStop(ctx, id, req.ServiceID); errStop != nil {
		return nil, errors.Wrapf(errStop, "Could not stop the service. service_id: %s", req.ServiceID)
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// v1ActiveflowsIDContinuePost handles
// /v1/activeflows/{id}/continue Post
func (h *listenHandler) v1ActiveflowsIDContinuePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1ActiveflowsIDContinuePost",
		"request": m,
	})

	// "/v1/activeflows/be2692f8-066a-11eb-847f-1b4de696fafb/continue"
	tmpVals := strings.Split(m.URI, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	var req request.V1DataActiveFlowsIDContinuePost
	if errUnmarshal := json.Unmarshal(m.Data, &req); errUnmarshal != nil {
		log.Errorf("Could not unmarshal the data. err: %v", errUnmarshal)
		return nil, errUnmarshal
	}

	go func() {
		if errContinue := h.activeflowHandler.ExecuteContinue(context.Background(), id, req.CurrentActionID); errContinue != nil {
			log.Errorf("Could not continue the activeflow correctly. err: %v", errContinue)
		}
	}()

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}
