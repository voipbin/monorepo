package dbhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-transfer-manager/models/transfer"
)

// ConferenceCreate creates a new conference record.
func (h *handler) TransferCreate(ctx context.Context, tr *transfer.Transfer) error {
	tr.TMCreate = h.utilHandler.TimeNow()
	tr.TMUpdate = nil
	tr.TMDelete = nil

	res := h.transferSetToCache(ctx, tr)

	return res

}

// transferGetFromCache returns transfer from the cache if possible.
func (h *handler) transferGetFromCache(ctx context.Context, id uuid.UUID) (*transfer.Transfer, error) {

	// get from cache
	res, err := h.cache.TransferGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// TransferSetToCache sets the given transfer to the cache
func (h *handler) transferSetToCache(ctx context.Context, conference *transfer.Transfer) error {
	if err := h.cache.TransferSet(ctx, conference); err != nil {
		return err
	}

	return nil
}

// TransferGet gets transfer.
func (h *handler) TransferGet(ctx context.Context, id uuid.UUID) (*transfer.Transfer, error) {

	res, err := h.transferGetFromCache(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// TransferGetByTransfererCallID gets transfer of the given transferer call id.
func (h *handler) TransferGetByTransfererCallID(ctx context.Context, transfererCallID uuid.UUID) (*transfer.Transfer, error) {

	res, err := h.cache.TransferGetByTransfererCallID(ctx, transfererCallID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// TransferGetByGroupcallID gets transfer of the given groupcall id.
func (h *handler) TransferGetByGroupcallID(ctx context.Context, groupcallID uuid.UUID) (*transfer.Transfer, error) {

	res, err := h.cache.TransferGetByGroupcallID(ctx, groupcallID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// TransferUpdate updates the given transfer.
func (h *handler) TransferUpdate(ctx context.Context, tr *transfer.Transfer) error {

	tr.TMUpdate = h.utilHandler.TimeNow()
	if errSet := h.cache.TransferSet(ctx, tr); errSet != nil {
		return errSet
	}

	return nil
}
