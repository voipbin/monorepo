package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

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
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res service.Service
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
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
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res service.Service
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
