package customerhandler

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/dbhandler"
)

// Freeze freezes the customer account.
func (h *customerHandler) Freeze(ctx context.Context, id uuid.UUID) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Freeze",
		"customer_id": id,
	})
	log.Debug("Freezing the customer account.")

	// Get customer, validate status
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get customer info. err: %v", err)
		return nil, err
	}

	// Idempotent: already frozen, return current state
	if c.Status == customer.StatusFrozen {
		log.Infof("The customer is already frozen. customer_id: %s", c.ID)
		return c, nil
	}

	// Cannot freeze a deleted customer
	if c.Status == customer.StatusDeleted || c.TMDelete != nil {
		log.Errorf("Cannot freeze a deleted customer. customer_id: %s", c.ID)
		return nil, fmt.Errorf("cannot freeze a deleted customer")
	}

	if err := h.db.CustomerFreeze(ctx, id); err != nil {
		// Handle race condition: if another request already froze this customer,
		// the DB returns ErrNotFound (0 rows affected). Re-fetch and return if frozen.
		if errors.Is(err, dbhandler.ErrNotFound) {
			refetched, refetchErr := h.Get(ctx, id)
			if refetchErr == nil && refetched.Status == customer.StatusFrozen {
				log.Infof("Concurrent freeze detected, returning already-frozen customer. customer_id: %s", id)
				return refetched, nil
			}
		}
		log.Errorf("Could not freeze the customer. err: %v", err)
		return nil, err
	}

	res, err := h.db.CustomerGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get frozen customer. err: %v", err)
		return nil, fmt.Errorf("could not get frozen customer")
	}

	// Publish customer_frozen event
	h.notifyHandler.PublishEvent(ctx, customer.EventTypeCustomerFrozen, res)

	return res, nil
}

// Recover recovers a frozen customer account.
func (h *customerHandler) Recover(ctx context.Context, id uuid.UUID) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Recover",
		"customer_id": id,
	})
	log.Debug("Recovering the customer account.")

	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get customer info. err: %v", err)
		return nil, err
	}

	if c.Status != customer.StatusFrozen {
		log.Errorf("Cannot recover a customer that is not frozen. status: %s", c.Status)
		return nil, fmt.Errorf("customer is not in frozen state")
	}

	if err := h.db.CustomerRecover(ctx, id); err != nil {
		// Handle race condition: if another request already recovered this customer,
		// the DB returns ErrNotFound (0 rows affected). Re-fetch and return if active.
		if errors.Is(err, dbhandler.ErrNotFound) {
			refetched, refetchErr := h.Get(ctx, id)
			if refetchErr == nil && refetched.Status == customer.StatusActive {
				log.Infof("Concurrent recover detected, returning already-active customer. customer_id: %s", id)
				return refetched, nil
			}
		}
		log.Errorf("Could not recover the customer. err: %v", err)
		return nil, err
	}

	res, err := h.db.CustomerGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get recovered customer. err: %v", err)
		return nil, fmt.Errorf("could not get recovered customer")
	}

	// Publish customer_recovered event
	h.notifyHandler.PublishEvent(ctx, customer.EventTypeCustomerRecovered, res)

	return res, nil
}
