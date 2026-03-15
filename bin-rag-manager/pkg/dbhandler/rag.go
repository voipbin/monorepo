package dbhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-rag-manager/models/rag"
)

// RagCreate inserts a new rag record
func (h *handler) RagCreate(ctx context.Context, r *rag.Rag) error {
	return fmt.Errorf("not implemented")
}

// RagGet retrieves a rag by ID
func (h *handler) RagGet(ctx context.Context, id uuid.UUID) (*rag.Rag, error) {
	return nil, fmt.Errorf("not implemented")
}

// RagGetsByCustomerID retrieves all rags for a customer
func (h *handler) RagGetsByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*rag.Rag, error) {
	return nil, fmt.Errorf("not implemented")
}

// RagUpdate updates rag fields by ID
func (h *handler) RagUpdate(ctx context.Context, id uuid.UUID, fields map[rag.Field]any) error {
	return fmt.Errorf("not implemented")
}

// RagDelete soft-deletes a rag by ID
func (h *handler) RagDelete(ctx context.Context, id uuid.UUID) error {
	return fmt.Errorf("not implemented")
}
