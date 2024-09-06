package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/sirupsen/logrus"

	"monorepo/bin-conference-manager/pkg/listenhandler/models/request"
)

// processV1ServicesTypeConferencecallPost handles POST /v1/services/type/conferencecall request
func (h *listenHandler) processV1ServicesTypeConferencecallPost(ctx context.Context, m *sock.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ServicesTypeConferencecallPost",
		"request": m,
	})

	var req request.V1DataServicesTypeConferencecallPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return nil, err
	}

	// start the service
	tmp, err := h.conferencecallHandler.ServiceStart(ctx, req.ConferenceID, req.ReferenceType, req.ReferenceID)
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
