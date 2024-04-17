package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
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

// AccountGet returns account info
func (h *handler) AccountGet(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	key := fmt.Sprintf("billing:account:%s", id)

	var res account.Account
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AccountGetByCustomerID returns account info of the given customerID
func (h *handler) AccountGetByCustomerID(ctx context.Context, customerID uuid.UUID) (*account.Account, error) {
	key := fmt.Sprintf("billing:account-customer_id:%s", customerID)

	var res account.Account
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AccountSet sets the account info into the cache.
func (h *handler) AccountSet(ctx context.Context, data *account.Account) error {
	key := fmt.Sprintf("billing:account:%s", data.ID)
	if err := h.setSerialize(ctx, key, data); err != nil {
		return err
	}

	keyCustomerID := fmt.Sprintf("billing:account-customer_id:%s", data.ID)
	if err := h.setSerialize(ctx, keyCustomerID, data); err != nil {
		return err
	}

	return nil
}

// BillingGet returns billing info
func (h *handler) BillingGet(ctx context.Context, id uuid.UUID) (*billing.Billing, error) {
	key := fmt.Sprintf("billing:billing:%s", id)

	var res billing.Billing
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// BillingGetByReferenceID returns billing info of the given reference id
func (h *handler) BillingGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*billing.Billing, error) {
	key := fmt.Sprintf("billing:billing-reference_id:%s", referenceID)

	var res billing.Billing
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// BillingSet sets the billing info into the cache.
func (h *handler) BillingSet(ctx context.Context, data *billing.Billing) error {
	key := fmt.Sprintf("billing:billing:%s", data.ID)
	if err := h.setSerialize(ctx, key, data); err != nil {
		return err
	}

	keyReferenceID := fmt.Sprintf("billing:billing-reference_id:%s", data.ReferenceID)
	if err := h.setSerialize(ctx, keyReferenceID, data); err != nil {
		return err
	}

	return nil
}
