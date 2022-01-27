package servicehandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
)

// AuthLogin generate jwt token of an user
func (h *serviceHandler) AuthLogin(username, password string) (string, error) {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "AuthLogin",
			"username": username,
			"password": len(password),
		},
	)

	u, err := h.reqHandler.UMV1UserLogin(ctx, 30, username, password)
	if err != nil {
		log.Warningf("Could not get user info. err: %v", err)
		return "", err
	}

	serialized := u.Serialize()
	token, err := middleware.GenerateToken("user", serialized)
	if err != nil {
		log.Errorf("Could not create a jwt token. err: %v", err)
		return "", fmt.Errorf("could not create a jwt token. err: %v", err)
	}

	return token, nil
}

// AuthCustomerLogin generate jwt token of a customer
func (h *serviceHandler) AuthLoginCustomer(username, password string) (string, error) {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "AuthCustomerLogin",
			"username": username,
			"password": len(password),
		},
	)

	u, err := h.reqHandler.CSV1Login(ctx, 30000, username, password)
	if err != nil {
		log.Warningf("Could not get user info. err: %v", err)
		return "", err
	}

	serialized := u.Serialize()
	token, err := middleware.GenerateToken("customer", serialized)
	if err != nil {
		log.Errorf("Could not create a jwt token. err: %v", err)
		return "", fmt.Errorf("could not create a jwt token. err: %v", err)
	}

	return token, nil
}
