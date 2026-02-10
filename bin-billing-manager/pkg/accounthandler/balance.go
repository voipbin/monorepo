package accounthandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
)

// IsValidBalanceByCustomerID returns false if the given customer's balance is not valid
func (h *accountHandler) IsValidBalanceByCustomerID(ctx context.Context, customerID uuid.UUID, billingType billing.ReferenceType, country string, count int) (bool, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "IsValidBalanceByCustomerID",
		"customer_id":  customerID,
		"billing_type": billingType,
		"country":      country,
	})

	a, err := h.GetByCustomerID(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get account info. err: %v", err)
		return false, errors.Wrap(err, "could not get account info")
	}

	res, err := h.IsValidBalance(ctx, a.ID, billingType, country, count)
	if err != nil {
		log.Errorf("Could not validate the account balance. err: %v", err)
		return false, errors.Wrap(err, "could not validate the account balance")
	}

	return res, nil
}

// IsValidBalance returns false if the given account's balance is not valid
func (h *accountHandler) IsValidBalance(ctx context.Context, accountID uuid.UUID, billingType billing.ReferenceType, country string, count int) (bool, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "IsValidBalance",
		"customer_id":  accountID,
		"billing_type": billingType,
		"country":      country,
	})

	a, err := h.Get(ctx, accountID)
	if err != nil {
		log.Errorf("Could not get account info. err: %v", err)
		return false, errors.Wrap(err, "could not get account info")
	}

	if a.TMDelete != nil {
		log.WithField("account", a).Debugf("The account has deleted already. account_id: %s", a.ID)
		return false, nil
	}

	if a.PlanType == account.PlanTypeUnlimited {
		return true, nil
	}

	// call_extension is always valid regardless of balance
	if billingType == billing.ReferenceTypeCallExtension {
		return true, nil
	}

	if count < 1 {
		count = 1
	}

	var expectCost float32
	switch billingType {
	case billing.ReferenceTypeNumber:
		expectCost = billing.DefaultCostPerUnitReferenceTypeNumber * float32(count)

	case billing.ReferenceTypeCall:
		expectCost = billing.DefaultCostPerUnitReferenceTypeCall * float32(count)

	case billing.ReferenceTypeSMS:
		expectCost = billing.DefaultCostPerUnitReferenceTypeSMS * float32(count)

	default:
		log.Errorf("Unsupported billing type. billing_type: %s", billingType)
		return false, fmt.Errorf("unsupported billing type")
	}

	if a.Balance > expectCost {
		return true, nil
	}
	log.Infof("The account has not enough balance. expect_cost: %f, balance: %f", expectCost, a.Balance)

	return false, nil
}
