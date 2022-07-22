package servicehandler

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
)

// WebsockCreate validates the tag's ownership and returns the message info.
func (h *serviceHandler) WebsockCreate(ctx context.Context, u *cscustomer.Customer, w http.ResponseWriter, r *http.Request) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "WebsockCreate",
			"customer_id": u.ID,
		},
	)

	if errRun := h.websockHandler.Run(ctx, w, r, u.ID); errRun != nil {
		log.Errorf("Could not run the websock handler correctly. err: %v", errRun)
		return errRun
	}

	return nil
}
