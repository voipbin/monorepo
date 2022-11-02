package servicehandler

import (
	"context"

	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	nmavailablenumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/availablenumber"
)

// AvailableNumberGets sends a handles available number get
// It sends a request to the number-manager to getting a list of calls.
// it returns list of available numbers if it succeed.
func (h *serviceHandler) AvailableNumberGets(ctx context.Context, u *cscustomer.Customer, size uint64, countryCode string) ([]*nmavailablenumber.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id":  u.ID,
		"username":     u.Username,
		"size":         size,
		"country_code": countryCode,
	})

	// get available numbers
	tmps, err := h.reqHandler.NumberV1AvailableNumberGets(ctx, u.ID, size, countryCode)
	if err != nil {
		log.Infof("Could not get available numbers info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*nmavailablenumber.WebhookMessage{}
	for _, tmp := range tmps {
		c := tmp.ConvertWebhookMessage()
		res = append(res, c)
	}

	return res, nil
}
