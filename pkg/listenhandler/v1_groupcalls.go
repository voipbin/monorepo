package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler/models/request"
)

// processV1GroupcallsPost handles POST /v1/groupcalls request
// It creates a new groupcall.
func (h *listenHandler) processV1GroupcallsPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CallsPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 2 {
		return simpleResponse(400), nil
	}

	var req request.V1DataGroupcallsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.groupcallHandler.Start(ctx, req.CustomerID, &req.Source, req.Destinations, req.FlowID, req.MasterCallID, req.RingMethod, req.AnswerMethod)
	if err != nil {
		log.Debugf("Could not create a outgoing call. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
