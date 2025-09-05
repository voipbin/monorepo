package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	amsummary "monorepo/bin-ai-manager/models/summary"
	amrequest "monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"
	"net/url"

	"github.com/gofrs/uuid"
)

// AIV1SummaryGets sends a request to ai-manager
// to getting a list of summaries info.
// it returns detail list of ais info if it succeed.
func (r *requestHandler) AIV1SummaryGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]amsummary.Summary, error) {
	uri := fmt.Sprintf("/v1/summaries?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/summaries", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res []amsummary.Summary
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// AIV1AICreate sends a request to ai-manager
// to creating a ai.
// it returns created ai if it succeed.
func (r *requestHandler) AIV1SummaryCreate(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	onEndFlowID uuid.UUID,
	referenceType amsummary.ReferenceType,
	referenceID uuid.UUID,
	language string,
	timeout int,
) (*amsummary.Summary, error) {
	uri := "/v1/summaries"

	data := &amrequest.V1DataSummariesPost{
		CustomerID: customerID,

		ActiveflowID: activeflowID,
		OnEndFlowID:  onEndFlowID,

		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		Language: language,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/summaries", timeout, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res amsummary.Summary
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1SummaryGet returns the ai.
func (r *requestHandler) AIV1SummaryGet(ctx context.Context, summaryID uuid.UUID) (*amsummary.Summary, error) {

	uri := fmt.Sprintf("/v1/summaries/%s", summaryID.String())

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/summaries/<summary-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amsummary.Summary
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1SummaryDelete sends a request to ai-manager
// to deleting a summary.
// it returns deleted ai if it succeed.
func (r *requestHandler) AIV1SummaryDelete(ctx context.Context, aiID uuid.UUID) (*amsummary.Summary, error) {
	uri := fmt.Sprintf("/v1/summaries/%s", aiID)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodDelete, "ai/summaries/<summary-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amsummary.Summary
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
