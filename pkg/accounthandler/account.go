package accounthandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/account"
)

// Get returns an account info
func (h *accountHandler) Get(ctx context.Context, customerID uuid.UUID) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Get",
		"customer_id": customerID,
	})

	res, err := h.db.AccountGet(ctx, customerID)
	if err == nil {
		return res, nil
	}

	tmp, err := h.reqHandler.CustomerV1CustomerGet(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get customer info. err: %v", err)
		return nil, err
	}

	// create and update the account
	res = account.CreateAccountFromCustomer(tmp)
	if errUpdate := h.Set(ctx, res); errUpdate != nil {
		// we couldn't update the account, but keep going because we've got customer info
		log.Errorf("Could not update the account. err: %v", errUpdate)
	}

	return res, nil
}

// Set sets the account info
func (h *accountHandler) Set(ctx context.Context, a *account.Account) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Set",
		"account": a,
	})

	if err := h.db.AccountSet(ctx, a); err != nil {
		log.Errorf("Could not set account. err: %v", err)
		return err
	}

	return nil
}

// UpdateByCustomer updates the account info by customer and return the updated account
func (h *accountHandler) UpdateByCustomer(ctx context.Context, m *cscustomer.Customer) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "UpdateByCustomer",
		"customer_id": m.ID,
	})

	res := account.CreateAccountFromCustomer(m)
	if errSet := h.Set(ctx, res); errSet != nil {
		// we couldn't update the account, but keep going because we've got customer info
		log.Errorf("Could not update the account. err: %v", errSet)
	}

	res, err := h.Get(ctx, m.ID)
	if err != nil {
		log.Errorf("Could not get updated account info. err: %v", err)
		return nil, err
	}
	log.WithField("account", res).Debugf("Updated account info. account_id: %s", res.ID)

	return res, nil
}
