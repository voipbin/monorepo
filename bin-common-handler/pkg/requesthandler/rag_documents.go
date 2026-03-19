package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-common-handler/models/sock"
	rmdocument "monorepo/bin-rag-manager/models/document"
)

// RagV1DocumentGet sends a request to rag-manager to get a document by ID.
func (r *requestHandler) RagV1DocumentGet(ctx context.Context, id uuid.UUID) (*rmdocument.Document, error) {
	uri := fmt.Sprintf("/v1/documents/%s", id)

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodGet, "rag/documents", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res rmdocument.Document
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RagV1DocumentGets sends a request to rag-manager to get a list of documents.
func (r *requestHandler) RagV1DocumentGets(ctx context.Context, pageToken string, pageSize uint64, filters map[rmdocument.Field]any) ([]*rmdocument.Document, error) {
	uri := fmt.Sprintf("/v1/documents?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodGet, "rag/documents", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []*rmdocument.Document
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}
