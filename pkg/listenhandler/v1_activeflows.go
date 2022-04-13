package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/listenhandler/models/request"
)

// v1ActiveflowsPost handles /v1/activeflows POST request
// creates a new activeflow with given data.
func (h *listenHandler) v1ActiveflowsPost(ctx context.Context, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	var reqData request.V1DataActiveFlowsPost
	if err := json.Unmarshal(req.Data, &reqData); err != nil {
		logrus.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// create active flow
	resActiveFlow, err := h.activeflowHandler.Create(ctx, reqData.ReferenceType, reqData.ReferenceID, reqData.FlowID)
	if err != nil {
		logrus.Errorf("Could not create a new active flow. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(resActiveFlow)
	if err != nil {
		logrus.Errorf("Could not marshal the res. err: %v", err)
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
func (h *listenHandler) v1ActiveflowsIDDelete(ctx context.Context, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	// "/v1/activeflows/be2692f8-066a-11eb-847f-1b4de696fafb"
	tmpVals := strings.Split(req.URI, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	log := logrus.WithFields(
		logrus.Fields{
			"func": "v1ActiveflowsIDDelete",
			"id":   id,
		},
	)
	log.Debug("Executing v1ActiveflowsIDDelete.")

	tmp, err := h.activeflowHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete activeflow. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		logrus.Errorf("Could not marshal the res. err: %v", err)
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
func (h *listenHandler) v1ActiveflowsIDNextGet(ctx context.Context, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	// "/v1/activeflows/be2692f8-066a-11eb-847f-1b4de696fafb/next"
	tmpVals := strings.Split(req.URI, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	var reqData request.V1DataActiveFlowsIDNextGet
	if err := json.Unmarshal(req.Data, &reqData); err != nil {
		logrus.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	resAction, err := h.activeflowHandler.GetNextAction(ctx, id, reqData.CurrentActionID)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("Found next action. call: %s, current_action_id: %s, next_action: %s", id, reqData.CurrentActionID, resAction)

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
func (h *listenHandler) v1ActiveflowsIDForwardActionIDPut(ctx context.Context, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	// "/v1/activeflows/be2692f8-066a-11eb-847f-1b4de696fafb/forward_action_id"
	tmpVals := strings.Split(req.URI, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	var reqData request.V1DataActiveFlowsIDForwardActionIDPut
	if err := json.Unmarshal(req.Data, &reqData); err != nil {
		logrus.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}
	log := logrus.WithFields(
		logrus.Fields{
			"func":              "v1ActiveFlowsIDForwardActionIDPut",
			"id":                id,
			"forward_action_id": reqData.ForwardActionID,
			"forward_now":       reqData.ForwardNow,
		},
	)
	log.Debug("Executing v1ActiveFlowsIDForwardActionIDPut.")

	if err := h.activeflowHandler.SetForwardActionID(ctx, id, reqData.ForwardActionID, reqData.ForwardNow); err != nil {
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
func (h *listenHandler) v1ActiveflowsIDExecutePost(ctx context.Context, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	// "/v1/activeflows/be2692f8-066a-11eb-847f-1b4de696fafb/execute"
	tmpVals := strings.Split(req.URI, "/")
	id := uuid.FromStringOrNil(tmpVals[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func": "v1ActiveFlowsIDExecutePost",
			"id":   id,
		},
	)
	log.Debug("Executing v1ActiveFlowsIDExecutePost.")

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
