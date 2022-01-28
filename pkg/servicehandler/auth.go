package servicehandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
)

// AuthLogin generate jwt token of an customer
func (h *serviceHandler) AuthLogin(username, password string) (string, error) {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "AuthLogin",
			"username": username,
			"password": len(password),
		},
	)

	u, err := h.reqHandler.CSV1Login(ctx, 30000, username, password)
	if err != nil {
		log.Warningf("Could not get customer info. err: %v", err)
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
