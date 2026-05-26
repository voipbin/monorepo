package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	amaiaudit "monorepo/bin-ai-manager/models/aiaudit"
	amrequest "monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"
)

// AIV1AIAuditCreate sends a request to ai-manager to trigger audit jobs for an aicall.
// It returns the list of created AIAudit records if it succeeds.
func (r *requestHandler) AIV1AIAuditCreate(ctx context.Context, customerID uuid.UUID, aicallID uuid.UUID, language string) ([]*amaiaudit.AIAudit, error) {
	uri := "/v1/aiaudits"

	data := &amrequest.V1DataAIAuditsPost{
		CustomerID: customerID,
		AIcallID:   aicallID,
		Language:   language,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/aiaudits", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []*amaiaudit.AIAudit
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// AIV1AIAuditList sends a request to ai-manager to get a paginated list of audit records.
// It returns a list of AIAudit records if it succeeds.
func (r *requestHandler) AIV1AIAuditList(ctx context.Context, pageToken string, pageSize uint64, filters map[amaiaudit.Field]any) ([]*amaiaudit.AIAudit, error) {
	uri := fmt.Sprintf("/v1/aiaudits?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/aiaudits", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []*amaiaudit.AIAudit
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// AIV1AIAuditGet sends a request to ai-manager to get a single audit record.
// It returns the AIAudit record if it succeeds.
func (r *requestHandler) AIV1AIAuditGet(ctx context.Context, id uuid.UUID) (*amaiaudit.AIAudit, error) {
	uri := fmt.Sprintf("/v1/aiaudits/%s", id.String())

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/aiaudits/<id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amaiaudit.AIAudit
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1AIAuditDelete sends a request to ai-manager to delete a single audit record.
// It returns the deleted AIAudit record if it succeeds.
func (r *requestHandler) AIV1AIAuditDelete(ctx context.Context, id uuid.UUID) (*amaiaudit.AIAudit, error) {
	uri := fmt.Sprintf("/v1/aiaudits/%s", id.String())

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodDelete, "ai/aiaudits/<id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amaiaudit.AIAudit
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
