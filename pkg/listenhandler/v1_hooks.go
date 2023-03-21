package listenhandler

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/listenhandler/models/request"
)

// processV1HooksPost handles
// POST /v1/hooks request
func (h *listenHandler) processV1HooksPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1HooksPost",
		"request": m,
	})

	var req request.V1DataHooksPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log.WithField("request", req).Debugf("Received hook request. request_uri: %s", req.ReceviedURI)

	if errHook := h.conversationHandler.Hook(ctx, req.ReceviedURI, req.ReceivedData); errHook != nil {
		log.Errorf("Could not hook the message correctly. err: %v", errHook)
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}
