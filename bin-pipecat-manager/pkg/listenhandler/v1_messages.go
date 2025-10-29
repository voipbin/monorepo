package listenhandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-pipecat-manager/pkg/listenhandler/models/request"

	"github.com/sirupsen/logrus"
)

// processV1MessagesPost handles /v1/messages POST request
func (h *listenHandler) processV1MessagesPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1MessagesPost",
		"request": m,
	})

	var req request.V1DataMessagesPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.pipecatcallHandler.SendMessage(ctx, req.PipecatcallID, req.MessageID, req.MessageText, req.RunImmediately, req.AudioResponse)
	if err != nil {
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
