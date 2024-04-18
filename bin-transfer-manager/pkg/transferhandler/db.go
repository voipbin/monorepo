package transferhandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transfer-manager/models/transfer"
)

// Create is handy function for creating a transfer.
func (h *transferHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	transferType transfer.Type,
	transfererCallID uuid.UUID,
	transfereeAddresses []commonaddress.Address,
	groupcallID uuid.UUID,
	confbridgeID uuid.UUID,
) (*transfer.Transfer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                 "Create",
		"customer_id":          customerID,
		"transferer_call_id":   transfererCallID,
		"transferee_addresses": transfereeAddresses,
		"groupcall_id":         groupcallID,
	})

	// create a conference struct
	res := &transfer.Transfer{
		ID:         h.utilHandler.UUIDCreate(),
		CustomerID: customerID,

		Type: transferType,

		TransfererCallID:    transfererCallID,
		TransfereeAddresses: transfereeAddresses,

		GroupcallID:  groupcallID,
		ConfbridgeID: confbridgeID,
	}

	if errCreate := h.db.TransferCreate(ctx, res); errCreate != nil {
		log.Errorf("Could not create tranfer. err: %v", errCreate)
		return nil, errors.Wrap(errCreate, "could not create transfer")
	}

	return res, nil
}

// Get returns transfer.
func (h *transferHandler) Get(ctx context.Context, id uuid.UUID) (*transfer.Transfer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Get",
		"transfer_id": id,
	})

	res, err := h.db.TransferGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get transfer. err: %v", err)
		return nil, errors.Wrap(err, "could not get transfer")
	}

	return res, nil
}

// GetByGroupcallID returns transfer of the given groupcall id.
func (h *transferHandler) GetByGroupcallID(ctx context.Context, groupcallID uuid.UUID) (*transfer.Transfer, error) {
	res, err := h.db.TransferGetByGroupcallID(ctx, groupcallID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get transfer")
	}

	return res, nil
}

// GetByTransfererCallID returns transfer of the given groupcall id.
func (h *transferHandler) GetByTransfererCallID(ctx context.Context, transfererCallID uuid.UUID) (*transfer.Transfer, error) {
	res, err := h.db.TransferGetByTransfererCallID(ctx, transfererCallID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get transfer")
	}

	return res, nil
}

// updateTransfereeCallID updates the transferee call id.
func (h *transferHandler) updateTransfereeCallID(ctx context.Context, id uuid.UUID, transfereeCallID uuid.UUID) (*transfer.Transfer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "updateTransfereeCallID",
		"transfer_id":        id,
		"transferee_call_id": transfereeCallID,
	})

	tr, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get transfer info. err: %v", err)
		return nil, errors.Wrap(err, "could not get transfer info")
	}

	tr.TransfereeCallID = transfereeCallID
	if errUpdate := h.db.TransferUpdate(ctx, tr); errUpdate != nil {
		log.Errorf("Could not update the transfer. err: %v", errUpdate)
		return nil, errors.Wrap(errUpdate, "could not update the transfer")
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get transfer info. err: %v", err)
		return nil, errors.Wrap(err, "could not get transfer info")
	}

	return res, nil
}
