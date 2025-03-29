package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	amaicall "monorepo/bin-ai-manager/models/aicall"
	cbrequest "monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
)

func (r *requestHandler) AIV1AIcallStart(ctx context.Context, activeflowID uuid.UUID, aiID uuid.UUID, referenceType amaicall.ReferenceType, referenceID uuid.UUID, gender amaicall.Gender, language string) (*amaicall.AIcall, error) {
	uri := "/v1/aicalls"

	data := &cbrequest.V1DataAIcallsPost{
		ActiveflowID: activeflowID,

		AIID: aiID,

		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		Gender:   gender,
		Language: language,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/aicalls", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res amaicall.AIcall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AIV1AIcallGets sends a request to ai-manager
// to getting a list of aicall info of the given customer id.
// it returns detail list of aicall info if it succeed.
func (r *requestHandler) AIV1AIcallGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]amaicall.AIcall, error) {
	uri := fmt.Sprintf("/v1/aicalls?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/aicalls", 30000, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []amaicall.AIcall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// AIV1AIcallGet returns the aicall.
func (r *requestHandler) AIV1AIcallGet(ctx context.Context, aicallID uuid.UUID) (*amaicall.AIcall, error) {

	uri := fmt.Sprintf("/v1/aicalls/%s", aicallID)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/aicalls/<aicall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not get conference. status: %d", tmp.StatusCode)
	}

	var res amaicall.AIcall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AIV1AIcallDelete sends a request to ai-manager
// to deleting a aicall.
// it returns deleted conference if it succeed.
func (r *requestHandler) AIV1AIcallDelete(ctx context.Context, aicallID uuid.UUID) (*amaicall.AIcall, error) {
	uri := fmt.Sprintf("/v1/aicalls/%s", aicallID)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodDelete, "ai/aicalls/<aicall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res amaicall.AIcall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
