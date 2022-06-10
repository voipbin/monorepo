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
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Get",
		},
	)

	res, err := h.db.AccountGet(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get account. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Set sets the account info
func (h *accountHandler) Set(ctx context.Context, a *account.Account) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Set",
		},
	)

	if err := h.db.AccountSet(ctx, a); err != nil {
		log.Errorf("Could not set account. err: %v", err)
		return err
	}

	return nil
}

// UpdateByCustomer updates the account info by customer and return the updated account
func (h *accountHandler) UpdateByCustomer(ctx context.Context, m *cscustomer.Customer) (*account.Account, error) {
	log := logrus.WithFields(
		logrus.Fields{
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
		log.Errorf("Could no")
	}

	return res, nil
}
