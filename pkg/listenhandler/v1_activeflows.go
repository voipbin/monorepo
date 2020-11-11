package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/listenhandler/request"
)

// v1ActiveFlowsPost handles /v1/active-flows POST request
// creates a new activeflow with given data.
func (h *listenHandler) v1ActiveFlowsPost(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	var reqData request.V1DataActiveFlowsPost
	if err := json.Unmarshal(req.Data, &reqData); err != nil {
		logrus.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// create active flow
	resActiveFlow, err := h.flowHandler.ActiveFlowCreate(ctx, reqData.CallID, reqData.FlowID)
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

// v1ActiveFlowsIDNextGet handles
// /v1/active-flows/{id}/next GET
// /v1/flows/{id}/actions/{id}/next GET
func (h *listenHandler) v1ActiveFlowsIDNextGet(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	// "/v1/active-flows/be2692f8-066a-11eb-847f-1b4de696fafb/next"
	tmpVals := strings.Split(req.URI, "/")
	callID := uuid.FromStringOrNil(tmpVals[3])

	var reqData request.V1DataActiveFlowsIDNextGet
	if err := json.Unmarshal(req.Data, &reqData); err != nil {
		logrus.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	resAction, err := h.flowHandler.ActiveFlowNextActionGet(ctx, callID, reqData.CurrentActionID)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("Found next action. call: %s, current_action_id: %s, next_action: %s", callID, reqData.CurrentActionID, resAction)

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
