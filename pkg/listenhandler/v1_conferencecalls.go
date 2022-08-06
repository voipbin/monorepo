package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/listenhandler/models/request"
)

// processV1ConferencecallsPost handles /v1/conferencecalls POST request
func (h *listenHandler) processV1ConferencecallsPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"handler": "processV1ConferencecallsPost",
			"uri":     m.URI,
			"data":    m.Data,
		},
	)

	var data request.V1DataConferencecallsPost
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return nil, err
	}

	// get conferernce info
	cf, err := h.conferenceHandler.Get(ctx, data.ConferenceID)
	if err != nil {
		log.Errorf("Could not find conference info. err: %v", err)
		return simpleResponse(400), nil
	}

	// create conferencecall
	cc, err := h.conferencecallHandler.Create(ctx, cf.CustomerID, cf.ID, data.ReferenceType, data.ReferenceID)
	if err != nil {
		log.Errorf("Could not create a conference. err: %v", err)
		return nil, err
	}
	log.WithField("conferencecall", cc).Debugf("Created a new conferencecall. conferencecall_id: %s", cc.ID)

	tmp, err := json.Marshal(cc)
	if err != nil {
		log.Errorf("Could not marshal the conference. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       tmp,
	}

	return res, nil
}

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
