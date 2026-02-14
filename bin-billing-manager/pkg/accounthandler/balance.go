package accounthandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/dbhandler"
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
		"account_id":   accountID,
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

	// Check token availability first, then fall back to credit balance.
	//
	// NOTE: For ReferenceTypeCall, we use an optimistic check. At validation time we don't
	// know the call's cost type (VN vs PSTN) — that is determined later from the call's
	// direction and address type. If tokens are available, we allow the call because it
	// could be a VN call covered by tokens. If the call turns out to be PSTN (credit-only)
	// and the account lacks credits, billing will fail at BillingEnd time with
	// ErrInsufficientBalance. This is the preferred trade-off: rejecting valid VN calls
	// when credit balance is low would be worse than allowing a PSTN call that later
	// fails billing.
	switch billingType {
	case billing.ReferenceTypeCall:
		// call could be VN (uses tokens) or PSTN (uses credits) — optimistic check
		cycle, err := h.allowanceHandler.GetCurrentCycle(ctx, accountID)
		if err != nil && err != dbhandler.ErrNotFound {
			log.Errorf("Could not get current cycle. err: %v", err)
			return false, errors.Wrap(err, "could not get current cycle")
		}

		if cycle != nil {
			remaining := cycle.TokensTotal - cycle.TokensUsed
			if remaining < 0 {
				remaining = 0
			}
			if remaining > 0 {
				// tokens available — call could be VN and covered by tokens
				promAccountBalanceCheckTotal.WithLabelValues("valid").Inc()
				return true, nil
			}
		}

		// no tokens available — check credit balance at the most expensive rate
		expectCost := billing.DefaultCreditPerUnitCallPSTNOutgoing * float32(count)
		if a.Balance > expectCost {
			promAccountBalanceCheckTotal.WithLabelValues("valid").Inc()
			return true, nil
		}

	case billing.ReferenceTypeSMS:
		cycle, err := h.allowanceHandler.GetCurrentCycle(ctx, accountID)
		if err != nil && err != dbhandler.ErrNotFound {
			log.Errorf("Could not get current cycle. err: %v", err)
			return false, errors.Wrap(err, "could not get current cycle")
		}

		if cycle != nil {
			remaining := cycle.TokensTotal - cycle.TokensUsed
			if remaining < 0 {
				remaining = 0
			}
			tokensNeeded := billing.DefaultTokenPerUnitSMS * count
			if remaining >= tokensNeeded {
				promAccountBalanceCheckTotal.WithLabelValues("valid").Inc()
				return true, nil
			}
		}

		// insufficient tokens — check credit balance
		expectCost := billing.DefaultCreditPerUnitSMS * float32(count)
		if a.Balance > expectCost {
			promAccountBalanceCheckTotal.WithLabelValues("valid").Inc()
			return true, nil
		}

	case billing.ReferenceTypeNumber, billing.ReferenceTypeNumberRenew:
		expectCost := billing.DefaultCreditPerUnitNumber * float32(count)
		if a.Balance > expectCost {
			promAccountBalanceCheckTotal.WithLabelValues("valid").Inc()
			return true, nil
		}

	default:
		log.Errorf("Unsupported billing type. billing_type: %s", billingType)
		return false, fmt.Errorf("unsupported billing type")
	}

	log.Infof("The account has not enough balance or tokens. balance: %f", a.Balance)
	promAccountBalanceCheckTotal.WithLabelValues("invalid").Inc()
	return false, nil
}
