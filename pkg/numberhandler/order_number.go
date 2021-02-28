package numberhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models"
)

// GetAvailableNumbers gets the numbers from the number providers
func (h *numberHandler) OrderNumbers(userID uint64, numbers []string) ([]*models.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"numbers": numbers,
		},
	)

	// send a request to number providers
	_, err := h.reqHandler.TelnyxNumberOrdersPost(numbers)
	if err != nil {
		log.Errorf("Could not send the order request to the telnyx. err: %v", err)
		return nil, err
	}

	// get order number details
	res := []*models.Number{}
	for _, number := range numbers {
		tmpNumber, err := h.createNumberByTelnyxOrderNumber(userID, number)
		if err != nil {
			log.Errorf("Could not handle the ordered number to the telnyx. number: %s, err: %v", number, err)
			continue
		}

		// append
		res = append(res, tmpNumber)
	}

	return res, nil
}

// createNumberByTelnyxOrderNumber creates a number by ordered number to the telnyx.
func (h *numberHandler) createNumberByTelnyxOrderNumber(userID uint64, number string) (*models.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"number": number,
		},
	)
	ctx := context.Background()

	// get number info
	numInfos, err := h.reqHandler.TelnyxPhoneNumbersGet(1, "", number)
	if err != nil {
		log.Errorf("Could not get correct number info. number: %s, err: %v", number, err)

		return nil, err
	}
	if len(numInfos) <= 0 {
		log.Errorf("Could not get number info. number: %s", number)
		return nil, err
	}

	// update connection id
	numInfo := numInfos[0]
	tmpNum, err := h.reqHandler.TelnyxPhoneNumbersIDUpdateConnectionID(numInfo.ID, ConnectionID)
	if err != nil {
		log.Errorf("Could not update connection ID info. err: %v", err)
		return nil, err
	}

	tmp := tmpNum.ConvertNumber()

	// add uuid
	tmp.ID = uuid.Must(uuid.NewV4())
	tmp.UserID = userID

	// insert into db
	if err := h.db.NumberCreate(ctx, tmp); err != nil {
		log.WithFields(
			logrus.Fields{
				"number": tmp,
			},
		).Errorf("Could not create a number. number: %s, err: %v", tmp.Number, err)
		return nil, err
	}

	res, err := h.db.NumberGet(ctx, tmp.ID)
	if err != nil {
		log.Errorf("Could not get created number info. err: %v", err)
		return nil, err
	}

	return res, err
}
