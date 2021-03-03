package servicehandler

import (
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
)

// AvailableNumberGets sends a handles available number get
// It sends a request to the number-manager to getting a list of calls.
// it returns list of available numbers if it succeed.
func (h *serviceHandler) AvailableNumberGets(u *models.User, size uint64, countryCode string) ([]*models.AvailableNumber, error) {
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
	res := []*models.AvailableNumber{}
	for _, tmp := range tmps {
		c := tmp.ConvertNumber()
		res = append(res, c)
	}

	return res, nil
}
