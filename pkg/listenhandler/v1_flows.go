package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/flowhandler/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/listenhandler/request"
	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/rabbitmq"
)

func (h *listenHandler) v1FlowsIDGet(req *rabbitmq.Request) (*rabbitmq.Response, error) {

	return nil, nil
}

// v1FlowsPost handles /v1/flows POST request
// creates a new flow with given data and return the created flow info.
func (h *listenHandler) v1FlowsPost(req *rabbitmq.Request) (*rabbitmq.Response, error) {
	ctx := context.Background()

	var reqData request.V1DataFlowPost
	if err := json.Unmarshal(req.Data, &reqData); err != nil {
		logrus.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	flow := &flow.Flow{
		ID:      reqData.ID,
		UserID:  reqData.UserID,
		Name:    reqData.Name,
		Detail:  reqData.Detail,
		Actions: reqData.Actions,

		TMCreate: getCurTime(),
	}

	// create flow
	resFlow, err := h.flowHandler.FlowCreate(ctx, flow, reqData.Persist)
	if err != nil {
		logrus.Errorf("Could not create anew flow. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(resFlow)
	if err != nil {
		logrus.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmq.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// handlerFlowsIDActionsIDGet handles
// /v1/flows/{id}/actions/{id} GET
func (h *listenHandler) v1FlowsIDActionsIDGet(req *rabbitmq.Request) (*rabbitmq.Response, error) {
	ctx := context.Background()

	// "/v1/flows/a6f4eae8-8a74-11ea-af75-3f1e61b9a236/actions/ab1f7732-8a74-11ea-98f6-9b02a042df6a"
	tmpVals := strings.Split(req.URI, "/")
	flowID := uuid.FromStringOrNil(tmpVals[3])
	actionID := uuid.FromStringOrNil(tmpVals[5])

	resAction, err := h.flowHandler.ActionGet(ctx, flowID, actionID)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resAction)
	if err != nil {
		return nil, err
	}

	res := &rabbitmq.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
