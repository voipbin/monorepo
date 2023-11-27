package servicehandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
)

// AuthLogin generate jwt token of an customer
func (h *serviceHandler) AuthLogin(ctx context.Context, username string, password string) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "AuthLogin",
		"username": username,
		"password": len(password),
	})

	// agent login
	a, err := h.reqHandler.AgentV1Login(ctx, 30000, username, password)
	if err != nil {
		log.Warningf("Could not login the agent. err: %v", err)
		return "", err
	}
	log.WithField("agent", a).Debugf("Found agent info. agent_id: %s, customer_id: %s", a.ID, a.CustomerID)

	// get customer info
	c, err := h.reqHandler.CustomerV1CustomerGet(ctx, a.CustomerID)
	if err != nil {
		log.Errorf("Could not get customer info. err: %v", err)
		return "", err
	}
	log.WithField("customer", c).Debugf("Found customer info. customer_id: %s", c.ID)

	data := map[string]interface{}{
		"customer": c,
		"agent":    a,
	}

	res, err := middleware.GenerateTokenWithData(data)
	if err != nil {
		log.Errorf("Could not create a jwt token. err: %v", err)
		return "", fmt.Errorf("could not create a jwt token. err: %v", err)
	}

	return res, nil
}
