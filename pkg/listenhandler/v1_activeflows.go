package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/listenhandler/models/request"
)

// v1ActiveflowsGet handles /v1/activeflows GET request
func (h *listenHandler) v1ActiveflowsGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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

	// parse the filters
	filters := h.utilHandler.URLParseFilters(u)

	tmp, err := h.activeflowHandler.Gets(ctx, pageToken, pageSize, filters)
	if err != nil {
		log.Errorf("Could not get activeflows. err: %v", err)
		return nil, errors.Wrap(err, "could not get activeflows")
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ActiveflowsPost handles /v1/activeflows POST request
// creates a new activeflow with given data.
func (h *listenHandler) v1ActiveflowsPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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
	resActiveFlow, err := h.activeflowHandler.Create(ctx, reqData.ID, reqData.ReferenceType, reqData.ReferenceID, reqData.FlowID)
	if err != nil {
		log.Errorf("Could not create a new active flow. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(resActiveFlow)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ActiveflowsIDGet handles
// /v1/activeflows/<activeflow-id> GET
func (h *listenHandler) v1ActiveflowsIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ActiveflowsIDDelete handles
// /v1/activeflows/{id} DELETE
func (h *listenHandler) v1ActiveflowsIDDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ActiveflowsIDNextGet handles
// /v1/activeflows/{id}/next GET
func (h *listenHandler) v1ActiveflowsIDNextGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ActiveflowsIDForwardActionIDPut handles
// /v1/activeflows/{id}/forward_action_id PUT
func (h *listenHandler) v1ActiveflowsIDForwardActionIDPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// v1ActiveflowsIDExecutePost handles
// /v1/activeflows/{id}/execute Post
func (h *listenHandler) v1ActiveflowsIDExecutePost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1ActiveflowsIDExecutePost",
		"request": m,
	})

	// "/v1/activeflows/be2692f8-066a-11eb-847f-1b4de696fafb/execute"
	tmpVals := strings.Split(m.URI, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	go func() {
		if err := h.activeflowHandler.Execute(ctx, id); err != nil {
			log.Errorf("Could not execute the active-flow correctly. err: %v", err)
		}
	}()

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// v1ActiveflowsIDStopPost handles
// /v1/activeflows/<activeflow-id>/stop Post
func (h *listenHandler) v1ActiveflowsIDStopPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ActiveflowsIDPushActionsPost handles
// /v1/activeflows/<activeflow-id>/push_actions Post
func (h *listenHandler) v1ActiveflowsIDPushActionsPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
