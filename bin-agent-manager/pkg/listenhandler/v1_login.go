package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/listenhandler/models/request"
)

// processV1Login handles Post /v1/login request
func (h *listenHandler) processV1Login(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 2 {
		return simpleResponse(400), nil
	}

	log := logrus.WithFields(logrus.Fields{
		"func": "processV1Login",
	})
	log.Debug("Executing processV1Login.")

	var reqData request.V1DataLoginPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.agentHandler.Login(ctx, reqData.Username, reqData.Password)
	if err != nil {
		log.Errorf("Could not login the agent info. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
