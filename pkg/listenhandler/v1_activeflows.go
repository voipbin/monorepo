package listenhandler

import (
	"context"
	"encoding/json"

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
