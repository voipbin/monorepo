package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	amai "monorepo/bin-ai-manager/models/ai"
	amrequest "monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
)

// AIV1AIGets sends a request to ai-manager
// to getting a list of ais info.
// it returns detail list of ais info if it succeed.
func (r *requestHandler) AIV1AIGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]amai.AI, error) {
	uri := fmt.Sprintf("/v1/ais?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/ais", 30000, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []amai.AI
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// AIV1AIGet returns the ai.
func (r *requestHandler) AIV1AIGet(ctx context.Context, aiID uuid.UUID) (*amai.AI, error) {

	uri := fmt.Sprintf("/v1/ais/%s", aiID.String())

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/ais/<ai-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not get conference. status: %d", tmp.StatusCode)
	}

	var res amai.AI
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AIV1AICreate sends a request to ai-manager
// to creating a ai.
// it returns created ai if it succeed.
func (r *requestHandler) AIV1AICreate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	engineType amai.EngineType,
	engineModel amai.EngineModel,
	engineData map[string]any,
	initPrompt string,
) (*amai.AI, error) {
	uri := "/v1/ais"

	data := &amrequest.V1DataAIsPost{
		CustomerID: customerID,
		Name:       name,
		Detail:     detail,

		EngineType:  engineType,
		EngineModel: engineModel,
		EngineData:  engineData,

		InitPrompt: initPrompt,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/ais", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res amai.AI
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AIV1AIDelete sends a request to ai-manager
// to deleting a ai.
// it returns deleted ai if it succeed.
func (r *requestHandler) AIV1AIDelete(ctx context.Context, aiID uuid.UUID) (*amai.AI, error) {
	uri := fmt.Sprintf("/v1/ais/%s", aiID)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodDelete, "ai/ais/<ai-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res amai.AI
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AIV1AIUpdate sends a request to ai-manager
// to updating an AI.
// it returns updated AI if it succeed.
func (r *requestHandler) AIV1AIUpdate(
	ctx context.Context,
	aiID uuid.UUID,
	name string,
	detail string,
	engineType amai.EngineType,
	engineModel amai.EngineModel,
	engineData map[string]any,
	initPrompt string,
) (*amai.AI, error) {
	uri := fmt.Sprintf("/v1/ais/%s", aiID)

	data := &amrequest.V1DataAIsIDPut{
		Name:   name,
		Detail: detail,

		EngineType:  engineType,
		EngineModel: engineModel,
		EngineData:  engineData,

		InitPrompt: initPrompt,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPut, "ai/ais/<ai-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res amai.AI
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
