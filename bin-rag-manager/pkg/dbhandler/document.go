package dbhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-rag-manager/models/document"
)

// DocumentCreate inserts a new document record
func (h *handler) DocumentCreate(ctx context.Context, d *document.Document) error {
	return fmt.Errorf("not implemented")
}

// DocumentGet retrieves a document by ID
func (h *handler) DocumentGet(ctx context.Context, id uuid.UUID) (*document.Document, error) {
	return nil, fmt.Errorf("not implemented")
}

// DocumentGetsByRagID retrieves all documents for a rag
func (h *handler) DocumentGetsByRagID(ctx context.Context, ragID uuid.UUID) ([]*document.Document, error) {
	return nil, fmt.Errorf("not implemented")
}

// DocumentGetsByCustomerID retrieves all documents for a customer
func (h *handler) DocumentGetsByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*document.Document, error) {
	return nil, fmt.Errorf("not implemented")
}

// DocumentUpdate updates document fields by ID
func (h *handler) DocumentUpdate(ctx context.Context, id uuid.UUID, fields map[document.Field]any) error {
	return fmt.Errorf("not implemented")
}

// DocumentDelete soft-deletes a document by ID
func (h *handler) DocumentDelete(ctx context.Context, id uuid.UUID) error {
	return fmt.Errorf("not implemented")
}

// DocumentDeleteByRagID soft-deletes all documents for a rag
func (h *handler) DocumentDeleteByRagID(ctx context.Context, ragID uuid.UUID) error {
	return fmt.Errorf("not implemented")
}
