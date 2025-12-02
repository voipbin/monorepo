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

	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/pkg/listenhandler/models/request"
)

// v1FlowsIDGet handles /v1/flows/{id} GET request
func (h *listenHandler) v1FlowsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1FlowsIDGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/flows/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	flowID := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.flowHandler.Get(ctx, flowID)
	if err != nil {
		log.Errorf("Could not get flow info. err: %v", err)
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

// v1FlowsIDPut handles /v1/flows/{id} PUT request
func (h *listenHandler) v1FlowsIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1FlowsIDPut",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/flows/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	flowID := uuid.FromStringOrNil(tmpVals[3])

	var req request.V1DataFlowsIDPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	tmp, err := h.flowHandler.Update(ctx, flowID, req.Name, req.Detail, req.Actions, req.OnCompleteFlowID)
	if err != nil {
		log.Errorf("Could not update the flow info. err: %v", err)
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

// v1FlowsIDDelete handles /v1/flows/{id} Delete request
func (h *listenHandler) v1FlowsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1FlowsIDDelete",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/flows/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	flowID := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.flowHandler.Delete(ctx, flowID)
	if err != nil {
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

// v1FlowsPost handles /v1/flows POST request
// creates a new flow with given data and return the created flow info.
func (h *listenHandler) v1FlowsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1FlowsPost",
		"request": m,
	})

	var req request.V1DataFlowsPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// create flow
	tmp, err := h.flowHandler.Create(
		ctx,
		req.CustomerID,
		req.Type,
		req.Name,
		req.Detail,
		req.Persist,
		req.Actions,
		req.OnCompleteFlowID,
	)
	if err != nil {
		log.Errorf("Could not create anew flow. err: %v", err)
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

// v1FlowsGet handles /v1/flows GET request
func (h *listenHandler) v1FlowsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1FlowsGet",
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

	filters, err := flow.ConvertStringMapToFieldMap(req)
	if err != nil {
		log.Errorf("Could not convert the filters. err: %v", err)
		return simpleResponse(400), nil
	}

	// gets the list of flows
	tmp, err := h.flowHandler.Gets(ctx, pageToken, pageSize, filters)
	if err != nil {
		log.Errorf("Could not get flows. err: %v", err)
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

// v1FlowsIDActionsPut handles /v1/flows/{id}/actions PUT request
func (h *listenHandler) v1FlowsIDActionsPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1FlowsIDActionsPut",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/flows/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	flowID := uuid.FromStringOrNil(tmpVals[3])

	var req request.V1DataFlowIDActionsPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	tmp, err := h.flowHandler.UpdateActions(ctx, flowID, req.Actions)
	if err != nil {
		log.Errorf("Could not update the flow actions. err: %v", err)
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

// v1FlowsIDActionsIDGet handles
// /v1/flows/{id}/actions/{id} GET
func (h *listenHandler) v1FlowsIDActionsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1FlowsIDActionsIDGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/flows/a6f4eae8-8a74-11ea-af75-3f1e61b9a236/actions/ab1f7732-8a74-11ea-98f6-9b02a042df6a"
	tmpVals := strings.Split(u.Path, "/")
	flowID := uuid.FromStringOrNil(tmpVals[3])
	actionID := uuid.FromStringOrNil(tmpVals[5])

	tmp, err := h.flowHandler.ActionGet(ctx, flowID, actionID)
	if err != nil {
		log.Errorf("Could not get actions. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
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
