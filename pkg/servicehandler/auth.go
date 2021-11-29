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

	u, err := h.reqHandler.UMV1UserLogin(ctx, username, password)
	if err != nil {
		log.Warningf("Could not get user info. err: %v", err)
		return "", err
	}

	serialized := u.Serialize()
	token, err := middleware.GenerateToken(serialized)
	if err != nil {
		log.Errorf("Could not create a jwt token. err: %v", err)
		return "", fmt.Errorf("could not create a jwt token. err: %v", err)
	}

	return token, nil
}
