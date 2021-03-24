package servicehandler

import (
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/availablenumber"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"

	"github.com/sirupsen/logrus"
)

// AvailableNumberGets sends a handles available number get
// It sends a request to the number-manager to getting a list of calls.
// it returns list of available numbers if it succeed.
func (h *serviceHandler) AvailableNumberGets(u *user.User, size uint64, countryCode string) ([]*availablenumber.AvailableNumber, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":         u.ID,
		"username":     u.Username,
		"size":         size,
		"country_code": countryCode,
	})

	// get available numbers
	tmps, err := h.reqHandler.NMAvailableNumbersGet(u.ID, size, countryCode)
	if err != nil {
		log.Infof("Could not get available numbers info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*availablenumber.AvailableNumber{}
	for _, tmp := range tmps {
		c := availablenumber.ConvertNumber(&tmp)
		res = append(res, c)
	}

	return res, nil
}
