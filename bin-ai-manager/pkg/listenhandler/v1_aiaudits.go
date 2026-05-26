package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/aiaudit"
	"monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

// processV1AIAuditsPost handles POST /v1/aiaudits
func (h *listenHandler) processV1AIAuditsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AIAuditsPost",
		"request": m,
	})

	var req request.V1DataAIAuditsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal request. err: %v", err)
		return simpleResponse(400), nil
	}

	records, err := h.aiauditHandler.Create(ctx, req.CustomerID, req.AIcallID, req.Language)
	if err != nil {
		log.Debugf("Could not create aiaudit. err: %v", err)
		errStr := err.Error()
		switch {
		case strings.Contains(errStr, "rate limit exceeded"):
			return simpleResponse(429), nil
		case strings.Contains(errStr, "already progressing"):
			return simpleResponse(409), nil
		case strings.Contains(errStr, "not terminated"):
			return simpleResponse(400), nil
		default:
			return errorResponse(err), nil
		}
	}

	data, err := json.Marshal(records)
	if err != nil {
		log.Errorf("Could not marshal response. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 202, DataType: "application/json", Data: data}, nil
}

// processV1AIAuditsGet handles GET /v1/aiaudits
func (h *listenHandler) processV1AIAuditsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AIAuditsGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		log.Errorf("Could not parse URI. err: %v", err)
		return simpleResponse(400), nil
	}

	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(m.Data)
	if err != nil {
		log.Errorf("Could not parse filters. err: %v", err)
		return simpleResponse(400), nil
	}

	typedFilters, err := utilhandler.ConvertFilters[aiaudit.FieldStruct, aiaudit.Field](aiaudit.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	list, err := h.aiauditHandler.List(ctx, pageSize, pageToken, typedFilters)
	if err != nil {
		log.Debugf("Could not list aiaudits. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(list)
	if err != nil {
		log.Errorf("Could not marshal response. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1AIAuditsIDGet handles GET /v1/aiaudits/<id>
func (h *listenHandler) processV1AIAuditsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AIAuditsIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Invalid aiaudit ID.")
		return simpleResponse(400), nil
	}

	record, err := h.aiauditHandler.Get(ctx, id)
	if err != nil {
		log.Debugf("Could not get aiaudit. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(record)
	if err != nil {
		log.Errorf("Could not marshal response. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1AIAuditsIDDelete handles DELETE /v1/aiaudits/<id>
func (h *listenHandler) processV1AIAuditsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AIAuditsIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Invalid aiaudit ID.")
		return simpleResponse(400), nil
	}

	record, err := h.aiauditHandler.Delete(ctx, id)
	if err != nil {
		log.Debugf("Could not delete aiaudit. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(record)
	if err != nil {
		log.Errorf("Could not marshal response. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}
