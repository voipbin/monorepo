package requesthandler

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-common-handler/models/sock"
	rmquery "monorepo/bin-rag-manager/models/query"
)

// RagV1RagQuery sends a query request to the rag-manager.
// It returns the query response with sources.
func (r *requestHandler) RagV1RagQuery(ctx context.Context, ragID uuid.UUID, queryText string, topK int) (*rmquery.Response, error) {
	uri := "/v1/query"

	req := struct {
		RagID uuid.UUID `json:"rag_id"`
		Query string    `json:"query"`
		TopK  int       `json:"top_k,omitempty"`
	}{
		RagID: ragID,
		Query: queryText,
		TopK:  topK,
	}

	m, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal request")
	}

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodPost, "rag/query", 30000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res rmquery.Response
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
