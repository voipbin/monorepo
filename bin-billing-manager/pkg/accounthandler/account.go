package accounthandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/account"
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

// UpdatePlanType updates the account's plan type
func (h *accountHandler) UpdatePlanType(ctx context.Context, id uuid.UUID, planType account.PlanType) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "UpdatePlanType",
		"id":        id,
		"plan_type": planType,
	})

	res, err := h.dbUpdatePlanType(ctx, id, planType)
	if err != nil {
		log.Errorf("Could not update the account plan type. err: %v", err)
		return nil, errors.Wrap(err, "could not update the account plan type")
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

// SetStatus sets the account's status
func (h *accountHandler) SetStatus(ctx context.Context, id uuid.UUID, status account.Status) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "SetStatus",
		"id":     id,
		"status": status,
	})

	// validate status
	switch status {
	case account.StatusActive, account.StatusFrozen, account.StatusDeleted:
		// valid
	default:
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	if err := h.db.AccountSetStatus(ctx, id, status); err != nil {
		log.Errorf("Could not set account status. err: %v", err)
		return nil, errors.Wrap(err, "could not set account status")
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated account. err: %v", err)
		return nil, errors.Wrap(err, "could not get updated account")
	}

	h.notifyHandler.PublishEvent(ctx, account.EventTypeAccountUpdated, res)

	return res, nil
}
