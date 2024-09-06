package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/pkg/listenhandler/models/request"
)

// processV1HooksPost handles
// POST /v1/hooks request
func (h *listenHandler) processV1HooksPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
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

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}
