package requesthandler

import (
	"context"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"

	amaiprompthistory "monorepo/bin-ai-manager/models/aiprompthistory"
	"monorepo/bin-common-handler/models/sock"
)

// AIV1AIPromptHistoryList returns the prompt history entries for the given AI.
func (r *requestHandler) AIV1AIPromptHistoryList(ctx context.Context, aiID uuid.UUID, pageToken string, pageSize uint64) ([]amaiprompthistory.AIPromptHistory, error) {
	uri := fmt.Sprintf("/v1/ais/%s/prompt_histories?page_token=%s&page_size=%d",
		aiID.String(), url.QueryEscape(pageToken), pageSize)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/ais/<ai-id>/prompt_histories", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res []amaiprompthistory.AIPromptHistory
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// AIV1AIPromptHistoryGet returns a single prompt history entry.
func (r *requestHandler) AIV1AIPromptHistoryGet(ctx context.Context, aiID uuid.UUID, historyID uuid.UUID) (*amaiprompthistory.AIPromptHistory, error) {
	uri := fmt.Sprintf("/v1/ais/%s/prompt_histories/%s", aiID.String(), historyID.String())

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/ais/<ai-id>/prompt_histories/<history-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amaiprompthistory.AIPromptHistory
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
