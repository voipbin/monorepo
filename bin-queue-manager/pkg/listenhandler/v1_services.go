package listenhandler

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/listenhandler/models/request"
)

// processV1ServicesTypeQueuecallPost handles POST /v1/services/type/queuecall request
func (h *listenHandler) processV1ServicesTypeQueuecallPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"handler": "processV1ServicesTypeQueuecallPost",
			"uri":     m.URI,
		},
	)

	var req request.V1DataServicesTypeQueuecallPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return nil, err
	}

	// start the service
	tmp, err := h.queuecallHandler.ServiceStart(ctx, req.QueueID, req.ActiveflowID, req.ReferenceType, req.ReferenceID, req.ExitActionID)
	if err != nil {
		log.Errorf("Could not create chatbotcall. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
