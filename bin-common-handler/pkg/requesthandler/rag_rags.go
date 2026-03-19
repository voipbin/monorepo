package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-common-handler/models/sock"
	rmquery "monorepo/bin-rag-manager/models/query"
	rmrag "monorepo/bin-rag-manager/models/rag"
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

// RagV1RagCreate sends a request to rag-manager to create a new rag.
func (r *requestHandler) RagV1RagCreate(ctx context.Context, customerID uuid.UUID, name, description string, storageFileIDs []uuid.UUID, sourceURLs []string) (*rmrag.Rag, error) {
	uri := "/v1/rags"

	m, err := json.Marshal(struct {
		CustomerID     uuid.UUID   `json:"customer_id"`
		Name           string      `json:"name"`
		Description    string      `json:"description"`
		StorageFileIDs []uuid.UUID `json:"storage_file_ids"`
		SourceURLs     []string    `json:"source_urls"`
	}{
		CustomerID:     customerID,
		Name:           name,
		Description:    description,
		StorageFileIDs: storageFileIDs,
		SourceURLs:     sourceURLs,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodPost, "rag/rags", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res rmrag.Rag
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RagV1RagAddSources sends a request to rag-manager to add sources to an existing rag.
func (r *requestHandler) RagV1RagAddSources(ctx context.Context, ragID uuid.UUID, storageFileIDs []uuid.UUID, sourceURLs []string) (*rmrag.Rag, error) {
	uri := fmt.Sprintf("/v1/rags/%s/sources", ragID)

	m, err := json.Marshal(struct {
		StorageFileIDs []uuid.UUID `json:"storage_file_ids"`
		SourceURLs     []string    `json:"source_urls"`
	}{
		StorageFileIDs: storageFileIDs,
		SourceURLs:     sourceURLs,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodPost, "rag/rags.sources", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res rmrag.Rag
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RagV1RagGet sends a request to rag-manager to get a rag by ID.
func (r *requestHandler) RagV1RagGet(ctx context.Context, id uuid.UUID) (*rmrag.Rag, error) {
	uri := fmt.Sprintf("/v1/rags/%s", id)

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodGet, "rag/rags", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res rmrag.Rag
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RagV1RagGets sends a request to rag-manager to get a list of rags.
func (r *requestHandler) RagV1RagGets(ctx context.Context, pageToken string, pageSize uint64, filters map[rmrag.Field]any) ([]*rmrag.Rag, error) {
	uri := fmt.Sprintf("/v1/rags?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodGet, "rag/rags", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []*rmrag.Rag
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// RagV1RagUpdate sends a request to rag-manager to update a rag.
func (r *requestHandler) RagV1RagUpdate(ctx context.Context, id uuid.UUID, fields map[rmrag.Field]any) (*rmrag.Rag, error) {
	uri := fmt.Sprintf("/v1/rags/%s", id)

	m, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodPut, "rag/rags", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res rmrag.Rag
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// RagV1RagDelete sends a request to rag-manager to delete a rag.
func (r *requestHandler) RagV1RagDelete(ctx context.Context, id uuid.UUID) error {
	uri := fmt.Sprintf("/v1/rags/%s", id)

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodDelete, "rag/rags", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return err
	}

	if tmp.StatusCode != 200 {
		return fmt.Errorf("failed to delete rag: status %d", tmp.StatusCode)
	}

	return nil
}
