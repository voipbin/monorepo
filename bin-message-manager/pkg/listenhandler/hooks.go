package listenhandler

import (
	"context"
	"encoding/json"
	"errors"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/pkg/listenhandler/models/request"
	"monorepo/bin-message-manager/pkg/messagehandler"
)

// processV1HooksPost handles POST /v1/hooks request
func (h *listenHandler) processV1HooksPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1HooksPost",
		"request": loggableRequest(m),
	})

	var req request.V1DataHooksPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Debugf("Could not unmarshal the data. data_size: %d, err: %v", len(m.Data), err)
		return simpleResponse(400), nil
	}

	if errHook := h.messageHandler.Hook(ctx, req.ReceivedURI, req.ReceivedData); errHook != nil {
		log.Errorf("Could not hook the message correctly. err: %v", errHook)

		// only a receipt that could not be stored is answered with an error, so that it can be delivered again. Every
		// other error is answered with success: a redelivered inbound message would be created twice.
		if errors.Is(errHook, messagehandler.ErrReceiptRetry) {
			return simpleResponse(500), nil
		}
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}
