package customerhandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
)

// Login validate the customer's username and password.
func (h *customerHandler) Login(ctx context.Context, username, password string) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Login",
		"username": username,
	})
	log.Debug("Customer login.")

	res, err := h.db.CustomerGetByUsername(ctx, username)
	if err != nil {
		log.Errorf("Could not get customer info. err: %v", err)
		return nil, fmt.Errorf("no user info")
	}

	if !h.helpHandler.HashCheck(password, res.PasswordHash) {
		return nil, fmt.Errorf("wrong password")
	}

	return res, nil
}
