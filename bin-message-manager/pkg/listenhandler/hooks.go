package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/pkg/listenhandler/models/request"
)

// processV1HooksPost handles POST /v1/hooks request
func (h *listenHandler) processV1HooksPost(ctx context.Context, m *sock.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1HooksPost",
		"request": m,
	})

	var req request.V1DataHooksPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	if errHook := h.messageHandler.Hook(ctx, req.ReceviedURI, req.ReceivedData); errHook != nil {
		log.Errorf("Could not hook the message correctly. err: %v", errHook)
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}
