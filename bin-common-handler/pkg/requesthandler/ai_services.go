package requesthandler

import (
	"context"
	"encoding/json"

	amaicall "monorepo/bin-ai-manager/models/aicall"
	aisummary "monorepo/bin-ai-manager/models/summary"
	airequest "monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/service"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
)

// AIV1ServiceTypeAIcallStart sends a request to ai-manager
// to starts a aicall service.
// it returns created service if it succeed.
func (r *requestHandler) AIV1ServiceTypeAIcallStart(
	ctx context.Context,
	aiID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType amaicall.ReferenceType,
	referenceID uuid.UUID,
	resume bool,
	gender amaicall.Gender,
	language string,
	requestTimeout int,
) (*service.Service, error) {
	uri := "/v1/services/type/aicall"

	data := &airequest.V1DataServicesTypeAIcallPost{
		AIID:          aiID,
		ActiveflowID:  activeflowID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
		Resume:        resume,
		Gender:        gender,
		Language:      language,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/services/type/aicall", requestTimeout, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res service.Service
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1ServiceTypeSummaryStart sends a request to ai-manager
// to starts a summary service.
// it returns created service if it succeed.
func (r *requestHandler) AIV1ServiceTypeSummaryStart(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	onEndFlowID uuid.UUID,
	referenceType aisummary.ReferenceType,
	referenceID uuid.UUID,
	language string,
	requestTimeout int,
) (*service.Service, error) {
	uri := "/v1/services/type/summary"

	data := &airequest.V1DataServicesTypeSummaryPost{
		CustomerID:    customerID,
		ActiveflowID:  activeflowID,
		OnEndFlowID:   onEndFlowID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
		Language:      language,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/services/type/summary", requestTimeout, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res service.Service
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1ServiceTypeTaskStart sends a request to ai-manager
// to starts a task service.
// it returns created service if it succeed.
func (r *requestHandler) AIV1ServiceTypeTaskStart(ctx context.Context, aiID uuid.UUID, activeflowID uuid.UUID) (*service.Service, error) {
	uri := "/v1/services/type/task"

	data := &airequest.V1DataServicesTypeTaskPost{
		AIID:         aiID,
		ActiveflowID: activeflowID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/services/type/task", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res service.Service
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
