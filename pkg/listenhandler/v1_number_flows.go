package listenhandler

import (
	"context"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processV1NumberFlowsPost handles DELETE /v1/number_flows request
func (h *listenHandler) processV1NumberFlowsDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1NumberFlowsDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	flowID := uuid.FromStringOrNil(uriItems[3])
	log.Debugf("Executing processV1OrderNumbersIDDelete. flow: %s", flowID)

	if err := h.numberHandler.RemoveNumbersFlowID(ctx, flowID); err != nil {
		log.Errorf("Could not remove flow id from the numbers. err: %v", err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
	}

	return res, nil
}
