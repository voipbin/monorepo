package addresshandler

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/contact"
)

// GetAddress returns a single address scoped to customerID.
func (h *addressHandler) GetAddress(ctx context.Context, customerID, addressID uuid.UUID) (*contact.Address, error) {
	return h.db.AddressGet(ctx, customerID, addressID)
}

// ListAddresses returns addresses for the customer with optional filters.
func (h *addressHandler) ListAddresses(ctx context.Context, customerID uuid.UUID, filters map[string]any, pageToken string, pageSize uint64) ([]contact.Address, error) {
	return h.db.AddressList(ctx, customerID, filters, pageToken, pageSize)
}
