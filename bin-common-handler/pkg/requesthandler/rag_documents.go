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

// RagV1DocumentCreate sends a request to rag-manager to create a new document.
func (r *requestHandler) RagV1DocumentCreate(ctx context.Context, customerID, ragID uuid.UUID, name string, docType rmdocument.DocType, sourceURL string, storageFileID uuid.UUID) (*rmdocument.Document, error) {
	uri := "/v1/documents"

	reqData := struct {
		CustomerID    uuid.UUID          `json:"customer_id"`
		RagID         uuid.UUID          `json:"rag_id"`
		Name          string             `json:"name"`
		DocType       rmdocument.DocType `json:"doc_type"`
		SourceURL     string             `json:"source_url,omitempty"`
		StorageFileID uuid.UUID          `json:"storage_file_id,omitempty"`
	}{
		CustomerID:    customerID,
		RagID:         ragID,
		Name:          name,
		DocType:       docType,
		SourceURL:     sourceURL,
		StorageFileID: storageFileID,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodPost, "rag/documents", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res rmdocument.Document
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

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

// RagV1DocumentDelete sends a request to rag-manager to delete a document.
func (r *requestHandler) RagV1DocumentDelete(ctx context.Context, id uuid.UUID) error {
	uri := fmt.Sprintf("/v1/documents/%s", id)

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodDelete, "rag/documents", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return err
	}

	if tmp.StatusCode != 200 {
		return fmt.Errorf("failed to delete document: status %d", tmp.StatusCode)
	}

	return nil
}
