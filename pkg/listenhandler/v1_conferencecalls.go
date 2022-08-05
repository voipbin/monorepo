package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processV1ConferencecallsIDGet handles /v1/conferencecalls/<id> GET request
func (h *listenHandler) processV1ConferencecallsIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "processV1ConferencecallsIDGet",
			"uri":  m.URI,
		},
	)

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	cc, err := h.conferencecallHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not remove the call from the conference. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := json.Marshal(cc)
	if err != nil {
		log.Errorf("Could not marshal the conferencecall. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       tmp,
	}

	return res, nil
}

// processV1ConferencecallsIDDelete handles /v1/conferencecalls/<id> DELETE request
func (h *listenHandler) processV1ConferencecallsIDDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "processV1ConferencecallsIDDelete",
			"uri":  m.URI,
		},
	)

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	cc, err := h.conferenceHandler.Leave(ctx, id)
	if err != nil {
		log.Errorf("Could not remove the call from the conference. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := json.Marshal(cc)
	if err != nil {
		log.Errorf("Could not marshal the conferencecall. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       tmp,
	}

	return res, nil
}
