package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/pkg/listenhandler/models/request"
)

// processV1AIsGet handles GET /v1/ais request
func (h *listenHandler) processV1AIsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AIsGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		log.Errorf("Could not parse the request uri. err: %v", err)
		return simpleResponse(400), nil
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
	typedFilters, err := utilhandler.ConvertFilters[ai.FieldStruct, ai.Field](ai.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	log = log.WithFields(logrus.Fields{
		"size":    pageSize,
		"token":   pageToken,
		"filters": typedFilters,
	})

	tmp, err := h.aiHandler.List(ctx, pageSize, pageToken, typedFilters)
	if err != nil {
		log.Debugf("Could not get conferences. err: %v", err)
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

// processV1AIsPost handles POST /v1/ais request
func (h *listenHandler) processV1AIsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1AIsPost",
		"request": m,
	})

	var req request.V1DataAIsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.aiHandler.Create(
		ctx,
		req.CustomerID,
		req.Name,
		req.Detail,
		req.EngineType,
		req.EngineModel,
		req.EngineData,
		req.EngineKey,
		req.InitPrompt,
		req.TTSType,
		req.TTSVoiceID,
		req.STTType,
		req.ToolNames,
	)
	if err != nil {
		log.Errorf("Could not create ai. err: %v", err)
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

// processV1AIsIDGet handles GET /v1/ais/<ai-id> request
func (h *listenHandler) processV1AIsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1AIsIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.aiHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get ai. err: %v", err)
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

// processV1AIsIDDelete handles DELETE /v1/ais/<ai-id> request
func (h *listenHandler) processV1AIsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1AIsIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.aiHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete ai. err: %v", err)
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

// processV1AIsIDPut handles PUT /v1/ais/<ai-id> request
func (h *listenHandler) processV1AIsIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1AIsIDPut",
		"request": m,
	})

	var req request.V1DataAIsIDPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return simpleResponse(400), nil
	}

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.aiHandler.Update(
		ctx,
		id,
		req.Name,
		req.Detail,
		req.EngineType,
		req.EngineModel,
		req.EngineData,
		req.EngineKey,
		req.InitPrompt,
		req.TTSType,
		req.TTSVoiceID,
		req.STTType,
		req.ToolNames,
	)
	if err != nil {
		log.Errorf("Could not update ai. err: %v", err)
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
