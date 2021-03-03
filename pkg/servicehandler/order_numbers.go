package servicehandler

import (
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
)

// OrderNumberGets sends a request to getting a list of ordered numbers
// It sends a request to the number-manager to getting a list of order_numbers.
// it returns list of order numbers if it succeed.
func (h *serviceHandler) OrderNumberGets(u *models.User, size uint64, token string) ([]*models.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"size":     size,
		"token":    "token",
	})

	// get available numbers
	tmps, err := h.reqHandler.NMOrderNumberGets(u.ID, token, size)
	if err != nil {
		log.Infof("Could not get available numbers info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*models.Number{}
	for _, tmp := range tmps {
		c := tmp.ConvertNumber()
		res = append(res, c)
	}

	return res, nil
}

// AvailableNumberGets sends a handles available number get
// It sends a request to the number-manager to getting a list of calls.
// it returns list of available numbers if it succeed.
func (h *serviceHandler) OrderNumberCreate(u *models.User, num string) (*models.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"numbers":  num,
	})

	// create numbers
	tmp, err := h.reqHandler.NMOrderNumberCreate(u.ID, num)
	if err != nil {
		log.Infof("Could not get available numbers info. err: %v", err)
		return nil, err
	}

	return tmp.ConvertNumber(), nil
}
