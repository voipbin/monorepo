package dbhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-webhook-manager/models/account"
)

// accountGetFromCache returns account from the cache.
func (h *handler) accountGetFromCache(ctx context.Context, id uuid.UUID) (*account.Account, error) {

	// get from cache
	res, err := h.cache.AccountGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// accountSetToCache sets the given account to the cache
func (h *handler) accountSetToCache(ctx context.Context, u *account.Account) error {
	if err := h.cache.AccountSet(ctx, u); err != nil {
		return err
	}

	return nil
}

// AccountGet returns the account.
func (h *handler) AccountGet(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	return h.accountGetFromCache(ctx, id)
}

// AccountSet returns sets the account
func (h *handler) AccountSet(ctx context.Context, u *account.Account) error {
	return h.accountSetToCache(ctx, u)
}
