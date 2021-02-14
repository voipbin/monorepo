package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/listenhandler/models/request"
)

// processV1ExtensionsPost handles /v1/extensions request
func (h *listenHandler) processV1ExtensionsPost(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 3 {
		return simpleResponse(400), nil
	}

	var reqData request.V1DataExtensionsPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		logrus.Debugf("Could not unmarshal the request data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	// create a new extension
	e, err := h.extensionHandler.CreateExtension(ctx, reqData.UserID, reqData.DomainID, reqData.Extension, reqData.Password)
	if err != nil {
		logrus.Errorf("Could not create a new extension correctly. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(e)
	if err != nil {
		logrus.Errorf("Could not marshal the response message. message: %v, err: %v", e, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
