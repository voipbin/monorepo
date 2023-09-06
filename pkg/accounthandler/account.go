package accounthandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"
)

// Create creates a new account and return the created account.
func (h *accountHandler) Create(ctx context.Context, customerID uuid.UUID, name string, detail string, paymentType account.PaymentType, payemntMethod account.PaymentMethod) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
	})

	res, err := h.dbCreate(ctx, customerID, name, detail, paymentType, payemntMethod)
	if err != nil {
		log.Errorf("Could not create the account. err: %v", err)
		return nil, errors.Wrap(err, "could not create the account")
	}

	return res, nil

}

// UpdateBasicInfo updates the account's basic info
func (h *accountHandler) UpdateBasicInfo(ctx context.Context, id uuid.UUID, name string, detail string) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "UpdateBasicInfo",
		"id":     id,
		"name":   name,
		"detail": detail,
	})

	res, err := h.dbUpdateBasicInfo(ctx, id, name, detail)
	if err != nil {
		log.Errorf("Could not update the account. err: %v", err)
		return nil, errors.Wrap(err, "could not update the account")
	}

	return res, nil
}

// UpdatePaymentInfo updates the account's basic info
func (h *accountHandler) UpdatePaymentInfo(ctx context.Context, id uuid.UUID, paymentType account.PaymentType, paymentMethod account.PaymentMethod) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "UpdatePaymentInfo",
		"id":     id,
		"name":   paymentType,
		"detail": paymentMethod,
	})

	res, err := h.dbUpdatePaymentInfo(ctx, id, paymentType, paymentMethod)
	if err != nil {
		log.Errorf("Could not update the account. err: %v", err)
		return nil, errors.Wrap(err, "could not update the account")
	}

	return res, nil
}
