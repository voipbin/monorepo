package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/listenhandler/models/request"
)

// processV1ConferencesGet handles GET /v1/conferences request
func (h *listenHandler) processV1ConferencesGet(ctx context.Context, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)
	conferenceType := u.Query().Get("type")

	// get customer id
	customerID := uuid.FromStringOrNil(u.Query().Get("customer_id"))
	log := logrus.WithFields(logrus.Fields{
		"customer_id": customerID,
		"size":        pageSize,
		"token":       pageToken,
	})

	log.Debug("Getting conference info.")
	confs, err := h.conferenceHandler.Gets(ctx, customerID, conference.Type(conferenceType), pageSize, pageToken)
	if err != nil {
		log.Debugf("Could not get conferences. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(confs)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", confs, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ConferencesPost handles /v1/conferences request
func (h *listenHandler) processV1ConferencesPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"handler": "processV1ConferencesPost",
			"uri":     m.URI,
			"data":    m.Data,
		},
	)

	var data request.V1DataConferencesPost
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return nil, err
	}

	// create conference
	cf, err := h.conferenceHandler.Create(ctx, conference.Type(data.Type), data.CustomerID, data.Name, data.Detail, data.Timeout, data.PreActions, data.PostActions)
	if err != nil {
		log.Errorf("Could not create a conference. err: %v", err)
		return nil, err
	}

	tmp, err := json.Marshal(cf)
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

// processV1ConferencesIDDelete handles /v1/conferences/<id> DELETE request
func (h *listenHandler) processV1ConferencesIDDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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

	if err := h.conferenceHandler.Terminate(ctx, id); err != nil {
		log.Errorf("Could not terminate the conference. err: %v", err)
		return simpleResponse(400), nil
	}

	return simpleResponse(200), nil
}

// processV1ConferencesIDPut handles /v1/conferences/<id> PUT request
func (h *listenHandler) processV1ConferencesIDPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"handler": "processV1ConferencesIDPut",
			"uri":     m.URI,
		},
	)

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var data request.V1DataConferencesIDPut
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return nil, err
	}

	cf, err := h.conferenceHandler.Update(ctx, id, data.Name, data.Detail, data.Timeout, data.PreActions, data.PostActions)
	if err != nil {
		log.Errorf("Could not update the conference. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := json.Marshal(cf)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       tmp,
	}

	return res, nil
}

// processV1ConferencesIDGet handles /v1/conferences/<id> GET request
func (h *listenHandler) processV1ConferencesIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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

	cf, err := h.conferenceHandler.Get(ctx, id)
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
		DataType:   "application/json",
		Data:       tmp,
	}

	return res, nil
}

// processV1ConferencesIDJoinPost handles /v1/conferences/<id>/join POST request
func (h *listenHandler) processV1ConferencesIDJoinPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "processV1ConferencesIDJoinPost",
			"uri":  m.URI,
		},
	)

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	cfID := uuid.FromStringOrNil(uriItems[3])

	var data request.V1DataConferencesIDJoinPost
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return nil, err
	}

	cc, err := h.conferenceHandler.Join(ctx, cfID, data.ReferenceType, data.ReferenceID)
	if err != nil {
		log.Errorf("Could not join the conference. err: %v", err)
		return nil, err
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

// processV1ConferencesIDRecordingIDPut handles /v1/conferences/<id>/recording_id PUT request
func (h *listenHandler) processV1ConferencesIDRecordingIDPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "processV1ConferencesIDRecordingIDPut",
			"uri":  m.URI,
		},
	)

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	cfID := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataConferencesIDRecordingIDPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return nil, err
	}

	tmp, err := h.conferenceHandler.UpdateRecordingID(ctx, cfID, req.RecordingID)
	if err != nil {
		log.Errorf("Could not update the conference recording id. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ConferencesIDRecordingStartPost handles /v1/conferences/<id>/recording_start POST request
func (h *listenHandler) processV1ConferencesIDRecordingStartPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "processV1ConferencesIDRecordingStartPost",
			"uri":  m.URI,
		},
	)

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	cfID := uuid.FromStringOrNil(uriItems[3])

	if errRecording := h.conferenceHandler.RecordingStart(ctx, cfID); errRecording != nil {
		log.Errorf("Could not start the conference recording id. err: %v", errRecording)
		return nil, errRecording
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// processV1ConferencesIDRecordingStopPost handles /v1/conferences/<id>/recording_stop POST request
func (h *listenHandler) processV1ConferencesIDRecordingStopPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "processV1ConferencesIDRecordingStopPost",
			"uri":  m.URI,
		},
	)

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	cfID := uuid.FromStringOrNil(uriItems[3])

	if errRecording := h.conferenceHandler.RecordingStop(ctx, cfID); errRecording != nil {
		log.Errorf("Could not stop the conference recording. err: %v", errRecording)
		return nil, errRecording
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// processV1ConferencesIDConferencecallIDsPost handles /v1/conferences/<conference-id>/conferencecall_ids POST request
func (h *listenHandler) processV1ConferencesIDConferencecallIDsPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "processV1ConferencesIDConferencecallIDsPost",
			"uri":  m.URI,
		},
	)

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	cfID := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataConferencesIDConferencecallIDsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return nil, err
	}

	tmp, err := h.conferenceHandler.AddConferencecallID(ctx, cfID, req.ConferencecallID)
	if err != nil {
		log.Errorf("Could not add the conferencecall id. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ConferencesIDConferencecallsConferencecallIDsIDDelete handles /v1/conferences/<conference-id>/conferencecall_ids/<conferencecall-id> DELETE request
func (h *listenHandler) processV1ConferencesIDConferencecallsConferencecallIDsIDDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "processV1ConferencesIDConferencecallsConferencecallIDsIDDelete",
			"uri":  m.URI,
		},
	)

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 6 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	cfID := uuid.FromStringOrNil(uriItems[3])
	ccID := uuid.FromStringOrNil(uriItems[5])

	tmp, err := h.conferenceHandler.RemoveConferencecallID(ctx, cfID, ccID)
	if err != nil {
		log.Errorf("Could not stop the conference recording. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
