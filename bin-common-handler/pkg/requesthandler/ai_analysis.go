package requesthandler

import (
	"context"
	"encoding/json"

	amanalysis "monorepo/bin-ai-manager/models/analysis"
	"monorepo/bin-common-handler/models/sock"
)

// AIV1ServiceTypeAnalysisRun sends a request to ai-manager's generic internal
// LLM gateway (POST /v1/services/type/analysis) and returns the structured
// response. requestTimeout is in milliseconds.
func (r *requestHandler) AIV1ServiceTypeAnalysisRun(ctx context.Context, req *amanalysis.Request, requestTimeout int) (*amanalysis.Response, error) {
	uri := "/v1/services/type/analysis"

	m, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/services/type/analysis", requestTimeout, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res amanalysis.Response
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
