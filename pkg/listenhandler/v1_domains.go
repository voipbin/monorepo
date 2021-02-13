package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/listenhandler/models/request"
)

// processV1DomainsPost handles /v1/domains request
func (h *listenHandler) processV1DomainsPost(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 3 {
		return simpleResponse(400), nil
	}

	var reqData request.V1DataDomainsPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		logrus.Debugf("Could not unmarshal the request data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	// create a new domain
	d, err := h.domainHandler.CreateDomain(ctx, int(reqData.UserID), reqData.DomainName)
	if err != nil {
		logrus.Errorf("Could not create a new domain correctly. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(d)
	if err != nil {
		logrus.Errorf("Could not marshal the response message. message: %v, err: %v", d, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
