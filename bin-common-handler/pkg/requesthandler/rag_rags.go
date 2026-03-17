package requesthandler

import (
	"context"
	"encoding/json"

	rmquery "monorepo/bin-rag-manager/models/query"
	"monorepo/bin-common-handler/models/sock"

	"github.com/pkg/errors"
)

// RagV1RagQuery sends a query request to the rag-manager.
// It returns the query response with answer and sources.
func (r *requestHandler) RagV1RagQuery(ctx context.Context, query string, docTypes []string, topK int) (*rmquery.Response, error) {
	uri := "/v1/rags/query"

	req := struct {
		Query    string   `json:"query"`
		DocTypes []string `json:"doc_types,omitempty"`
		TopK     int      `json:"top_k,omitempty"`
	}{
		Query:    query,
		DocTypes: docTypes,
		TopK:     topK,
	}

	m, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal request")
	}

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodPost, "rag/rags/query", 30000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res rmquery.Response
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
