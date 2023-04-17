package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	uuid "github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/transfer-manager.git/models/transfer"
)

// getSerialize returns cached serialized info.
func (h *handler) getSerialize(ctx context.Context, key string, data interface{}) error {
	tmp, err := h.Cache.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(tmp), &data); err != nil {
		return err
	}
	return nil
}

// setSerialize sets the info into the cache after serialization.
func (h *handler) setSerialize(ctx context.Context, key string, data interface{}) error {
	tmp, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if err := h.Cache.Set(ctx, key, tmp, time.Hour*24).Err(); err != nil {
		return err
	}
	return nil
}

// TransferGet returns cached transfer info
func (h *handler) TransferGet(ctx context.Context, id uuid.UUID) (*transfer.Transfer, error) {
	key := fmt.Sprintf("transfer:transfer:%s", id)

	var res transfer.Transfer
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// TransferSet sets the transfer info into the cache.
func (h *handler) TransferSet(ctx context.Context, tr *transfer.Transfer) error {
	key := fmt.Sprintf("transfer:transfer:%s", tr.ID)
	if err := h.setSerialize(ctx, key, tr); err != nil {
		return err
	}

	keyTransfererCallID := fmt.Sprintf("transfer:transferer_call_id:%s", tr.TransfererCallID)
	if err := h.setSerialize(ctx, keyTransfererCallID, tr); err != nil {
		return err
	}

	keyGroupcallID := fmt.Sprintf("transfer:groupcall_id:%s", tr.GroupcallID)
	if err := h.setSerialize(ctx, keyGroupcallID, tr); err != nil {
		return err
	}

	return nil
}

// TransferGetByTransfererCallID returns cached transfer info of the given transferer call id
func (h *handler) TransferGetByTransfererCallID(ctx context.Context, transfererCallID uuid.UUID) (*transfer.Transfer, error) {
	key := fmt.Sprintf("transfer:transferer_call_id:%s", transfererCallID)

	var res transfer.Transfer
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// TransferGetByGroupcallID returns cached transfer info of the given groupcall id
func (h *handler) TransferGetByGroupcallID(ctx context.Context, groupcallID uuid.UUID) (*transfer.Transfer, error) {
	key := fmt.Sprintf("transfer:groupcall_id:%s", groupcallID)

	var res transfer.Transfer
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
