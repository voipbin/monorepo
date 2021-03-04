package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler/models/request"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processV1ConferencesPost handles /v1/conferences request
func (h *listenHandler) processV1ConferencesPost(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"handler": "processV1ConferencesPost",
			"uri":     m.URI,
			"data":    m.Data,
		},
	)

	var data request.V1DataConferencesIDPost
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return nil, err
	}

	// create a request conference
	reqConf := &conference.Conference{
		UserID:  data.UserID,
		Type:    data.Type,
		Name:    data.Name,
		Detail:  data.Detail,
		Timeout: data.Timeout,
		Data:    data.Data,
	}

	// create a conference
	cf, err := h.conferenceHandler.Start(reqConf)
	if err != nil {
		log.Errorf("Could not start the conference. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := json.Marshal(cf)
	if err != nil {
		log.Errorf("Could not marshal the conference. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		Data:       tmp,
	}

	return res, nil
}

// processV1ConferencesIDDelete handles /v1/conferences/<id> DELETE request
func (h *listenHandler) processV1ConferencesIDDelete(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"handler": "processV1ConferencesIDDelete",
			"uri":     m.URI,
		},
	)

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var data request.V1DataConferencesIDDelete
	if m.Data != nil {
		if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
			return nil, err
		}
	} else {
		data.Reason = ""
	}

	if err := h.conferenceHandler.Terminate(id, data.Reason); err != nil {
		log.Errorf("Could not terminate the conference. err: %v", err)
		return simpleResponse(400), nil
	}

	return simpleResponse(200), nil
}

// processV1ConferencesIDGet handles /v1/conferences/<id> GET request
func (h *listenHandler) processV1ConferencesIDGet(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"handler": "processV1ConferencesIDGet",
			"uri":     m.URI,
		},
	)

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	cf, err := h.db.ConferenceGet(context.Background(), id)
	if err != nil {
		log.Errorf("Could not get conference info. conference: %s, err: %v", id, err)
		return simpleResponse(400), nil
	}

	tmp, err := json.Marshal(cf)
	if err != nil {
		log.Errorf("Could not marshal the conference info. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		Data:       tmp,
	}

	return res, nil
}

// processV1ConferencesIDCallsIDDelete handles /v1/conferences/<id>/calls/<id> DELETE request
func (h *listenHandler) processV1ConferencesIDCallsIDDelete(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"handler": "processV1ConferencesIDCallsIDDelete",
			"uri":     m.URI,
		},
	)

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 6 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	cfID := uuid.FromStringOrNil(uriItems[3])
	callID := uuid.FromStringOrNil(uriItems[5])

	if cfID == uuid.Nil || callID == uuid.Nil {
		log.Errorf("Wrong id info. conference: %s, call: %s", cfID, callID)
		return simpleResponse(400), nil
	}

	if err := h.conferenceHandler.Leave(cfID, callID); err != nil {
		log.Errorf("Could not remove the call from the conference. err: %v", err)
		return simpleResponse(400), nil
	}

	return simpleResponse(200), nil
}
