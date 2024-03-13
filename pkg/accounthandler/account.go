package accounthandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/account"
)

// Get returns given customer id's account.
func (h *accountHandler) Get(ctx context.Context, customerID uuid.UUID) (*account.Account, error) {
	log := logrus.WithFields(
		logrus.Fields{
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
	if errUpdate := h.Update(ctx, res); errUpdate != nil {
		// we couldn't update the account, but keep going because we've got customer info
		log.Errorf("Could not update the account. err: %v", errUpdate)
	}

	return res, nil
}

// Update updates the account
func (h *accountHandler) Update(ctx context.Context, m *account.Account) error {
	return h.db.AccountSet(ctx, m)
}

// UpdateByCustomer updates the account by customer and return the updated account
func (h *accountHandler) UpdateByCustomer(ctx context.Context, m *cscustomer.Customer) (*account.Account, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "UpdateByCustomer",
			"customer_id": m.ID,
		})

	res := account.CreateAccountFromCustomer(m)
	if errUpdate := h.Update(ctx, res); errUpdate != nil {
		// we couldn't update the account, but keep going because we've got customer info
		log.Errorf("Could not update the account. err: %v", errUpdate)
	}

	return res, nil
}
