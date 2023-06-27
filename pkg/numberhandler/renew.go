package numberhandler

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
)

// RenewNumbers renew the numbers
func (h *numberHandler) RenewNumbers(ctx context.Context, tmRenew string) ([]*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "RenewNumbers",
		"tm_renew": tmRenew,
	})

	// get list of numbers
	numbers, err := h.GetsByTMRenew(ctx, tmRenew)
	if err != nil {
		log.Errorf("Could not get list of numbers. err: %v", err)
		return nil, errors.Wrap(err, "could not get list of numbers")
	}

	// renew the numbers
	var res []*number.Number
	for _, n := range numbers {
		log.WithField("number", n).Debugf("Renewing the number. number_id: %s, number: %s", n.ID, n.Number)
		tmp, err := h.UpdateRenew(ctx, n.ID)
		if err != nil {
			log.Errorf("Could not update the number's renew info. err: %v", err)
			continue
		}
		log.WithField("number", n).Debugf("Renewed the number info. number_id: %s, number: %s", n.ID, n.Number)
		res = append(res, tmp)
	}

	return res, nil
}
