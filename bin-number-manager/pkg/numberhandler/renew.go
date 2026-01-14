package numberhandler

import (
	"context"
	"fmt"
	"time"

	bmbilling "monorepo/bin-billing-manager/models/billing"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-number-manager/models/number"
)

// RenewNumbers renew the numbers
func (h *numberHandler) RenewNumbers(ctx context.Context, days int, hours int, tmRenew string) ([]*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "RenewNumbers",
		"days":     days,
		"hours":    hours,
		"tm_renew": tmRenew,
	})

	var res []*number.Number
	var err error
	switch {
	case days != 0:
		res, err = h.renewNumbersByDays(ctx, days)
	case hours != 0:
		res, err = h.renewNumbersByHours(ctx, hours)
	case tmRenew != "":
		res, err = h.renewNumbersByTMRenew(ctx, tmRenew)
	default:
		log.Errorf("Could not find correct renew time")
		return nil, fmt.Errorf("could not find correct renew time")
	}

	if err != nil {
		log.Errorf("Could not renew the numbers. err: %v", err)
		return nil, errors.Wrap(err, "could not renew the numbers")
	}

	return res, nil
}

// renewNumbersByTMRenew renew the numbers by tm_renew
func (h *numberHandler) renewNumbersByTMRenew(ctx context.Context, tmRenew string) ([]*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "renewNumbersByTMRenew",
		"tm_renew": tmRenew,
	})

	// get list of numbers
	numbers, err := h.dbListByTMRenew(ctx, tmRenew)
	if err != nil {
		log.Errorf("Could not get list of numbers. err: %v", err)
		return nil, errors.Wrap(err, "could not get list of numbers")
	}

	// renew the numbers
	var res []*number.Number
	for _, n := range numbers {

		valid, err := h.reqHandler.CustomerV1CustomerIsValidBalance(ctx, n.CustomerID, bmbilling.ReferenceTypeNumber, "us", 1)
		if err != nil {
			log.Errorf("Could not validate the customer balance. err: %v", err)
			continue
		}

		if !valid {
			log.WithField("number", n).Errorf("The customer has not enough balance for number renew.")
			tmp, err := h.Delete(ctx, n.ID)
			if err != nil {
				log.Errorf("Could not release the number. err: %v", err)
				continue
			}
			log.WithField("number", tmp).Debugf("Deleted number.")
		}

		log.WithField("number", n).Debugf("Renewing the number. number_id: %s, number: %s", n.ID, n.Number)

		fields := map[number.Field]any{
			number.FieldTMRenew: h.utilHandler.TimeGetCurTime(),
		}
		tmp, err := h.dbUpdate(ctx, n.ID, fields, number.EventTypeNumberRenewed)
		if err != nil {
			log.Errorf("Could not update the number's renew info. err: %v", err)
			continue
		}
		log.WithField("number", n).Debugf("Renewed the number info. number_id: %s, number: %s", n.ID, n.Number)
		res = append(res, tmp)
	}

	return res, nil
}

// renewNumbersByDays renew the numbers by tm_renew
func (h *numberHandler) renewNumbersByDays(ctx context.Context, days int) ([]*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "renewNumbersByDays",
		"days": days,
	})

	tmRenew := h.utilHandler.TimeGetCurTimeAdd(-(time.Hour * 24 * time.Duration(days)))
	log.Debugf("Renwing numbers. tm_renew: %s", tmRenew)

	res, err := h.renewNumbersByTMRenew(ctx, tmRenew)
	if err != nil {
		log.Errorf("Could not renew the numbers. err: %v", err)
		return nil, errors.Wrap(err, "could not renew the numbers")
	}

	return res, nil
}

// renewNumbersByDays renew the numbers by tm_renew
func (h *numberHandler) renewNumbersByHours(ctx context.Context, hours int) ([]*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "renewNumbersByHours",
		"hours": hours,
	})

	tmRenew := h.utilHandler.TimeGetCurTimeAdd(-(time.Hour * time.Duration(hours)))
	log.Debugf("Renwing numbers. tm_renew: %s", tmRenew)

	res, err := h.renewNumbersByTMRenew(ctx, tmRenew)
	if err != nil {
		log.Errorf("Could not renew the numbers. err: %v", err)
		return nil, errors.Wrap(err, "could not renew the numbers")
	}

	return res, nil
}
