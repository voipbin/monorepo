package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processV1ChatbotcallsGet handles GET /v1/chatbotcall request
func (h *listenHandler) processV1ChatbotcallsGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"handler": "processV1ChatbotcallsGet",
			"uri":     m.URI,
		},
	)

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get customer id
	customerID := uuid.FromStringOrNil(u.Query().Get("customer_id"))
	log = log.WithFields(logrus.Fields{
		"customer_id": customerID,
		"size":        pageSize,
		"token":       pageToken,
	})

	tmp, err := h.chatbotcallHandler.Gets(ctx, customerID, pageSize, pageToken)
	if err != nil {
		log.Debugf("Could not get conferences. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
