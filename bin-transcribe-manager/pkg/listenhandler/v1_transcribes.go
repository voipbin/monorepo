package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/pkg/listenhandler/models/request"
)

// processV1TranscribesPost handles POST /v1/transcribes request
// It creates a new transcribe.
func (h *listenHandler) processV1TranscribesPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1TranscribesPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 3 {
		return simpleResponse(400), nil
	}

	var req request.V1DataTranscribesPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.transcribeHandler.Start(ctx, req.CustomerID, req.ActiveflowID, req.OnEndFlowID, req.ReferenceType, req.ReferenceID, req.Language, req.Direction)
	if err != nil {
		log.Debugf("Could not create a transcribe. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1TranscribesGet handles GET /v1/transcribes request
func (h *listenHandler) processV1TranscribesGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1TranscribesGet",
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

	// get filters from request body
	tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(m.Data)
	if err != nil {
		log.Errorf("Could not parse filters. err: %v", err)
		return simpleResponse(400), nil
	}

	// convert to typed filters
	filters, err := utilhandler.ConvertFilters[transcribe.FieldStruct, transcribe.Field](transcribe.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.transcribeHandler.Gets(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get transcribes. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1TranscribesIDGet handles GET /v1/transcribes/<id> request
func (h *listenHandler) processV1TranscribesIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1TranscribesIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	c, err := h.transcribeHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get transcribe. err: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(c)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1TranscribesIDDelete handles Delete /v1/transcribes/<id> request
func (h *listenHandler) processV1TranscribesIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1TranscribesIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.transcribeHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the transcribe. err: %v", err)
		return simpleResponse(400), nil
	}

	d, err := json.Marshal(tmp)
	if err != nil {
		logrus.Errorf("Could not marshal the data. err: %v", err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       d,
	}

	return res, nil
}

// processV1TranscribesIDStopPost handles /v1/transcribes/<id>/stop POST request
func (h *listenHandler) processV1TranscribesIDStopPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1TranscribesIDStopPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	tr, err := h.transcribeHandler.Stop(ctx, id)
	if err != nil {
		log.Errorf("Could not stop the transcribe. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := json.Marshal(tr)
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

// processV1TranscribesIDHealthCheckPost handles /v1/transcribes/<transcribe-id>/health-check request
func (h *listenHandler) processV1TranscribesIDHealthCheckPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1TranscribesIDHealthCheckPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataTranscribesIDHealthCheckPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not marshal the request message. message: %v, err: %v", req, err)
		return nil, err
	}

	h.transcribeHandler.HealthCheck(ctx, id, req.RetryCount)
	return nil, nil
}
