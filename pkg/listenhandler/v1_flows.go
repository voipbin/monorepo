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
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/listenhandler/request"
)

// v1FlowsIDGet handles /v1/flows/{id} GET request
func (h *listenHandler) v1FlowsIDGet(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/flows/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	flowID := uuid.FromStringOrNil(tmpVals[3])

	flow, err := h.flowHandler.FlowGet(ctx, flowID)
	if err != nil {
		logrus.Errorf("Could not get flow info. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(flow)
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

// v1FlowsIDPut handles /v1/flows/{id} PUT request
func (h *listenHandler) v1FlowsIDPut(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/flows/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	flowID := uuid.FromStringOrNil(tmpVals[3])

	var reqData request.V1DataFlowIDPut
	if err := json.Unmarshal(req.Data, &reqData); err != nil {
		logrus.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// create a update flow
	f := &flow.Flow{
		ID:      flowID,
		Name:    reqData.Name,
		Detail:  reqData.Detail,
		Actions: reqData.Actions,
	}

	flow, err := h.flowHandler.FlowUpdate(ctx, f)
	if err != nil {
		logrus.Errorf("Could not update the flow info. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(flow)
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

// v1FlowsIDDelete handles /v1/flows/{id} Delete request
func (h *listenHandler) v1FlowsIDDelete(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/flows/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	flowID := uuid.FromStringOrNil(tmpVals[3])

	if err := h.flowHandler.FlowDelete(ctx, flowID); err != nil {
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
	}

	return res, nil
}

// v1FlowsPost handles /v1/flows POST request
// creates a new flow with given data and return the created flow info.
func (h *listenHandler) v1FlowsPost(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1FlowsGet handles /v1/flows GET request
func (h *listenHandler) v1FlowsGet(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	u, err := url.Parse(req.URI)
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

	resFlows, err := h.flowHandler.FlowGetsByUserID(ctx, userID, pageToken, pageSize)
	if err != nil {
		logrus.Errorf("Could not get flows. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(resFlows)
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

// handlerFlowsIDActionsIDGet handles
// /v1/flows/{id}/actions/{id} GET
func (h *listenHandler) v1FlowsIDActionsIDGet(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/flows/a6f4eae8-8a74-11ea-af75-3f1e61b9a236/actions/ab1f7732-8a74-11ea-98f6-9b02a042df6a"
	tmpVals := strings.Split(u.Path, "/")
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

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
