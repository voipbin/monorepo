package addresshandler

//go:generate mockgen -package addresshandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// AddressHandler handles read-only operations on contact addresses.
// Write operations (AddAddress, UpdateAddress, RemoveAddress) remain in
// ContactHandler because they publish contact events and require contact
// context.
type AddressHandler interface {
	// GetAddress returns a single address scoped to customerID.
	GetAddress(ctx context.Context, customerID, addressID uuid.UUID) (*contact.Address, error)

	// ListAddresses returns addresses for the customer with optional filters.
	// filters keys: "contact_id" (uuid.UUID), "type" (string).
	ListAddresses(ctx context.Context, customerID uuid.UUID, filters map[string]any, pageToken string, pageSize uint64) ([]contact.Address, error)
}

type addressHandler struct {
	db dbhandler.DBHandler
}

// NewAddressHandler returns an AddressHandler backed by the given DBHandler.
func NewAddressHandler(db dbhandler.DBHandler) AddressHandler {
	return &addressHandler{db: db}
}
