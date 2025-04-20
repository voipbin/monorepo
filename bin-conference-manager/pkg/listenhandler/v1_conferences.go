package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/pkg/listenhandler/models/request"
)

// processV1ConferencesGet handles GET /v1/conferences request
func (h *listenHandler) processV1ConferencesGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConferencesGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get filters
	filters := h.utilHandler.URLParseFilters(u)

	confs, err := h.conferenceHandler.Gets(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Debugf("Could not get conferences. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(confs)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", confs, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ConferencesPost handles /v1/conferences request
func (h *listenHandler) processV1ConferencesPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConferencesPost",
		"request": m,
	})

	var req request.V1DataConferencesPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return simpleResponse(400), nil
	}

	// create conference
	cf, err := h.conferenceHandler.Create(
		ctx,
		req.ID,
		req.CustomerID,
		conference.Type(req.Type),
		req.Name,
		req.Detail,
		req.Data,
		req.Timeout,
		req.PreFlowID,
		req.PostFlowID,
	)
	if err != nil {
		log.Errorf("Could not create a conference. err: %v", err)
		return simpleResponse(500), nil
	}

	tmp, err := json.Marshal(cf)
	if err != nil {
		log.Errorf("Could not marshal the conference. err: %v", err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       tmp,
	}

	return res, nil
}

// processV1ConferencesIDDelete handles /v1/conferences/<id> DELETE request
func (h *listenHandler) processV1ConferencesIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConferencesIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	cf, err := h.conferenceHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the conference. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := json.Marshal(cf)
	if err != nil {
		log.Errorf("Could not marshal the conference. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       tmp,
	}

	return res, nil
}

// processV1ConferencesIDPut handles /v1/conferences/<id> PUT request
func (h *listenHandler) processV1ConferencesIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConferencesIDPut",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataConferencesIDPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return nil, err
	}

	cf, err := h.conferenceHandler.Update(
		ctx,
		id,
		req.Name,
		req.Detail,
		req.Data,
		req.Timeout,
		req.PreFlowID,
		req.PostFlowID,
	)
	if err != nil {
		log.Errorf("Could not update the conference. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := json.Marshal(cf)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       tmp,
	}

	return res, nil
}

// processV1ConferencesIDGet handles /v1/conferences/<id> GET request
func (h *listenHandler) processV1ConferencesIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConferencesIDGet",
		"request": m,
	})

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

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       tmp,
	}

	return res, nil
}

// processV1ConferencesIDRecordingIDPut handles /v1/conferences/<id>/recording_id PUT request
func (h *listenHandler) processV1ConferencesIDRecordingIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConferencesIDRecordingIDPut",
		"request": m,
	})

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

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ConferencesIDRecordingStartPost handles /v1/conferences/<id>/recording_start POST request
func (h *listenHandler) processV1ConferencesIDRecordingStartPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConferencesIDRecordingStartPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	cfID := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataConferencesIDRecordingStartPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return nil, err
	}

	tmp, err := h.conferenceHandler.RecordingStart(ctx, cfID, req.ActiveflowID, req.Format, req.Duration, req.OnEndFlowID)
	if err != nil {
		log.Errorf("Could not start the conference recording id. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ConferencesIDRecordingStopPost handles /v1/conferences/<id>/recording_stop POST request
func (h *listenHandler) processV1ConferencesIDRecordingStopPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConferencesIDRecordingStopPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	cfID := uuid.FromStringOrNil(uriItems[3])

	tmp, errRecording := h.conferenceHandler.RecordingStop(ctx, cfID)
	if errRecording != nil {
		log.Errorf("Could not stop the conference recording. err: %v", errRecording)
		return nil, errRecording
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ConferencesIDTranscribeStartPost handles /v1/conferences/<conference-id>/transcribe_start POST request
func (h *listenHandler) processV1ConferencesIDTranscribeStartPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConferencesIDTranscribeStartPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	cfID := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataConferencesIDTranscribeStartPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return nil, err
	}

	tmp, err := h.conferenceHandler.TranscribeStart(ctx, cfID, req.Language)
	if err != nil {
		log.Errorf("Could not start the conference transcribe. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ConferencesIDTranscribeStopPost handles /v1/conferences/<conference-id>/transcribe_stop POST request
func (h *listenHandler) processV1ConferencesIDTranscribeStopPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConferencesIDTranscribeStopPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	cfID := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.conferenceHandler.TranscribeStop(ctx, cfID)
	if err != nil {
		log.Errorf("Could not stop the conference transcribe. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ConferencesIDStopPost handles /v1/conferences/<id>/stop POST request
func (h *listenHandler) processV1ConferencesIDStopPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConferencesIDStopPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	cfID := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.conferenceHandler.Terminating(ctx, cfID)
	if err != nil {
		log.Errorf("Could not start the conference recording id. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
