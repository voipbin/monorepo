package listenhandler

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/listenhandler/models/request"
)

// processV1SetupPost handles POST /v1/setup request
func (h *listenHandler) processV1SetupPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "processV1SetupPost",
		},
	)
	log.WithField("request", m).Debug("Received request.")

	var req request.V1DataSetupPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	if errSetup := h.conversationHandler.Setup(ctx, req.CustomerID, req.ReferenceType); errSetup != nil {
		log.Errorf("Could not setup. err: %v", errSetup)
		return nil, errSetup
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}
