package accounthandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"
)

// dbCreate creates a new account and return the created account.
func (h *accountHandler) dbCreate(ctx context.Context, customerID uuid.UUID, name string, detail string, paymentType account.PaymentType, payemntMethod account.PaymentMethod) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "dbCreate",
		"customer_id": customerID,
	})

	id := h.utilHandler.UUIDCreate()
	a := &account.Account{
		ID:            id,
		CustomerID:    customerID,
		Name:          name,
		Detail:        detail,
		Type:          account.TypeNormal,
		Balance:       0,
		PaymentType:   paymentType,
		PaymentMethod: payemntMethod,
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

	cs, err := h.reqHandler.CustomerV1CustomerGet(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get customer info")
		return nil, errors.Wrap(err, "could not get customer info")
	}

	res, err := h.db.AccountGet(ctx, cs.BillingAccountID)
	if err != nil {
		log.Errorf("Could not get account. err: %v", err)
		return nil, errors.Wrap(err, "could not get account info")
	}

	return res, nil
}

// Gets returns list of accounts.
func (h *accountHandler) Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "Gets",
		"size":  size,
		"token": token,
	})

	res, err := h.db.AccountGets(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get accounts. err: %v", err)
		return nil, errors.Wrap(err, "could not get accounts info")
	}

	return res, nil
}

// SubtractBalance substracts the balance of the given customer id.
func (h *accountHandler) SubtractBalance(ctx context.Context, accountID uuid.UUID, balance float32) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SubtractBalance",
		"customer_id": accountID,
		"balance":     balance,
	})

	if errSub := h.db.AccountSubtractBalance(ctx, accountID, balance); errSub != nil {
		log.Errorf("Could not subsctract the balance. err: %v", errSub)
		return nil, errors.Wrap(errSub, "could not subsctract the balance")
	}

	res, err := h.db.AccountGet(ctx, accountID)
	if err != nil {
		log.Errorf("Could not get updated account. err: %v", err)
		return nil, errors.Wrap(err, "could not get updated account info")
	}

	return res, nil
}

// AddBalance adds the balance of the given customer id.
func (h *accountHandler) AddBalance(ctx context.Context, accountID uuid.UUID, balance float32) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AddBalance",
		"customer_id": accountID,
		"balance":     balance,
	})

	if errSub := h.db.AccountAddBalance(ctx, accountID, balance); errSub != nil {
		log.Errorf("Could not add the balance. err: %v", errSub)
		return nil, errors.Wrap(errSub, "could not add the balance")
	}

	res, err := h.db.AccountGet(ctx, accountID)
	if err != nil {
		log.Errorf("Could not get updated account. err: %v", err)
		return nil, errors.Wrap(err, "could not get updated account info")
	}

	return res, nil
}

// Delete deletes the given account
func (h *accountHandler) Delete(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Delete",
		"account_id": id,
	})

	if errDelete := h.db.AccountDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not delete the account. err: %v", errDelete)
		return nil, errors.Wrap(errDelete, "could not delete the account")
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted account. err: %v", err)
		return nil, errors.Wrap(err, "could not get deleted account")
	}

	return res, nil
}

// dbUpdateBasicInfo updates the account's basic info
func (h *accountHandler) dbUpdateBasicInfo(ctx context.Context, id uuid.UUID, name string, detail string) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "UpdateBasicInfo",
		"id":     id,
		"name":   name,
		"detail": detail,
	})

	if errSet := h.db.AccountSet(ctx, id, name, detail); errSet != nil {
		log.Errorf("Could not update the account. err: %v", errSet)
		return nil, errors.Wrap(errSet, "could not update the account")
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated account. err: %v", err)
		return nil, errors.Wrap(err, "could not get updated account")
	}

	return res, nil
}

// dbUpdatePaymentInfo updates the account's payment info
func (h *accountHandler) dbUpdatePaymentInfo(ctx context.Context, id uuid.UUID, paymentType account.PaymentType, paymentMethod account.PaymentMethod) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "dbUpdatePaymentInfo",
		"id":             id,
		"payment_type":   paymentType,
		"payment_method": paymentMethod,
	})

	if errSet := h.db.AccountSetPaymentInfo(ctx, id, paymentType, paymentMethod); errSet != nil {
		log.Errorf("Could not update the account. err: %v", errSet)
		return nil, errors.Wrap(errSet, "could not update the account")
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated account. err: %v", err)
		return nil, errors.Wrap(err, "could not get updated account")
	}

	return res, nil
}
