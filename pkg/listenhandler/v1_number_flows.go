package listenhandler

import (
	"context"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processV1NumberFlowsPost handles DELETE /v1/number_flows request
func (h *listenHandler) processV1NumberFlowsDelete(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	uriItems := strings.Split(req.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	flowID := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"flow": flowID,
		})
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
