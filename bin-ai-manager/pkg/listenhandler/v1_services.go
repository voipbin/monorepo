package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/pkg/listenhandler/models/request"
)

// processV1ServicesTypeAIcallPost handles POST /v1/services/type/aicall request
func (h *listenHandler) processV1ServicesTypeAIcallPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1ServicesTypeAIcallPost",
		"request": m,
	})

	var req request.V1DataServicesTypeAIcallPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.aicallHandler.ServiceStart(ctx, req.AIID, req.ActiveflowID, req.ReferenceType, req.ReferenceID, req.Gender, req.Language)
	if err != nil {
		log.Errorf("Could not start ai service. err: %v", err)
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

// processV1ServicesTypeSummaryPost handles POST /v1/services/type/summary request
func (h *listenHandler) processV1ServicesTypeSummaryPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1ServicesTypeSummaryPost",
		"request": m,
	})

	var req request.V1DataServicesTypeSummaryPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.summaryHandler.ServiceStart(
		ctx,
		req.CustomerID,
		req.ActiveflowID,
		req.OnEndFlowID,
		req.ReferenceType,
		req.ReferenceID,
		req.Language,
	)
	if err != nil {
		log.Errorf("Could not start summary service. err: %v", err)
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

// processV1ServicesTypeTaskPost handles POST /v1/services/type/task request
func (h *listenHandler) processV1ServicesTypeTaskPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1ServicesTypeTaskPost",
		"request": m,
	})

	var req request.V1DataServicesTypeTaskPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.aicallHandler.ServiceStartTypeTask(ctx, req.AIID, req.ActiveflowID)
	if err != nil {
		log.Errorf("Could not start task service. err: %v", err)
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
