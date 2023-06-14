package accounthandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/billing-manager.git/pkg/dbhandler"
)

// IsValidBalanceByCustomerID returns false if the given customer's balance is not valid
func (h *accountHandler) IsValidBalanceByCustomerID(ctx context.Context, customerID uuid.UUID) (bool, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "IsValidBalanceByCustomerID",
		"customer_id": customerID,
	})

	a, err := h.GetByCustomerID(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get account info. err: %v", err)
		return false, errors.Wrap(err, "could not get account info")
	}

	if a.TMDelete < dbhandler.DefaultTimeStamp {
		log.WithField("account", a).Debugf("The account has deleted already. account_id: %s", a.ID)
		return false, nil
	}

	if a.Type == account.TypeAdmin {
		return true, nil
	}

	if a.Balance > 0 {
		return true, nil
	}

	return false, nil
}
