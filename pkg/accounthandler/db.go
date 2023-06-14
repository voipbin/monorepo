package accounthandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"
)

// Create creates a new account and return the created account.
func (h *accountHandler) Create(ctx context.Context, customerID uuid.UUID) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
	})

	id := h.utilHandler.UUIDCreate()
	a := &account.Account{
		ID:            id,
		CustomerID:    customerID,
		Name:          "",
		Detail:        "",
		Type:          account.TypeNormal,
		Balance:       0,
		PaymentType:   account.PaymentTypeNone,
		PaymentMethod: account.PaymentMethodNone,
	}

	if errCreate := h.db.AccountCreate(ctx, a); errCreate != nil {
		log.Errorf("Could not create a billing. err: %v", errCreate)
	}
	promAccountCreateTotal.Inc()

	res, err := h.db.AccountGet(ctx, a.ID)
	if err != nil {
		log.Errorf("Could not get a created billing. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, account.EventTypeAccountCreated, res)

	return res, nil
}

// Get returns a account.
func (h *accountHandler) Get(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Get",
		"account_id": id,
	})

	res, err := h.db.AccountGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get account. err: %v", err)
		return nil, errors.Wrap(err, "could not get account info")
	}

	return res, nil
}

// GetByCustomerID returns a account of the given customer id.
func (h *accountHandler) GetByCustomerID(ctx context.Context, customerID uuid.UUID) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "GetByCustomerID",
		"customer_id": customerID,
	})

	res, err := h.db.AccountGetByCustomerID(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get account. err: %v", err)
		return nil, errors.Wrap(err, "could not get account info")
	}

	return res, nil
}

// SubstractBalanceByCustomer substracts the balance of the given customer id.
func (h *accountHandler) SubstractBalanceByCustomer(ctx context.Context, customerID uuid.UUID, balance float32) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SubstractBalanceByCustomer",
		"customer_id": customerID,
		"balance":     balance,
	})

	if errSub := h.db.AccountSubstractBalanceByCustomerID(ctx, customerID, balance); errSub != nil {
		log.Errorf("Could not subsctract the balance. err: %v", errSub)
		return nil, errors.Wrap(errSub, "could not subsctract the balance")
	}

	res, err := h.db.AccountGetByCustomerID(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get updated account. err: %v", err)
		return nil, errors.Wrap(err, "could not get updated account info")
	}

	return res, nil
}

// AddBalanceByCustomer adds the balance of the given customer id.
func (h *accountHandler) AddBalanceByCustomer(ctx context.Context, customerID uuid.UUID, balance float32) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AddBalanceByCustomer",
		"customer_id": customerID,
		"balance":     balance,
	})

	if errSub := h.db.AccountAddBalanceByCustomerID(ctx, customerID, balance); errSub != nil {
		log.Errorf("Could not add the balance. err: %v", errSub)
		return nil, errors.Wrap(errSub, "could not add the balance")
	}

	res, err := h.db.AccountGetByCustomerID(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get updated account. err: %v", err)
		return nil, errors.Wrap(err, "could not get updated account info")
	}

	return res, nil
}
