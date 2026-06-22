package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/analysis"
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

	tmp, err := h.aicallHandler.ServiceStart(ctx, req.AssistanceType, req.AssistanceID, req.ActiveflowID, req.ReferenceType, req.ReferenceID)
	if err != nil {
		log.Errorf("Could not start ai service. err: %v", err)
		return errorResponse(err), nil
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
		return errorResponse(err), nil
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

// processV1ServicesTypeAnalysisPost handles POST /v1/services/type/analysis request.
//
// This is the generic internal-only LLM gateway: it takes a prompt + data + JSON
// schema and returns the schema-conformant structured JSON. The request and
// response are the analysis domain types directly (pass-through internal contract).
func (h *listenHandler) processV1ServicesTypeAnalysisPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1ServicesTypeAnalysisPost",
		"request": m,
	})

	var req analysis.Request
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.analysisHandler.Run(ctx, &req)
	if err != nil {
		log.Errorf("Could not run the analysis gateway. err: %v", err)
		return errorResponse(err), nil
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

	tmp, err := h.aicallHandler.ServiceStartTypeTask(ctx, req.AssistanceType, req.AssistanceID, req.ActiveflowID)
	if err != nil {
		log.Errorf("Could not start task service. err: %v", err)
		return errorResponse(err), nil
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
