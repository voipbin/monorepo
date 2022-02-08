package numberhandlertelnyx

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
)

// CreateOrderNumbers creates a new order numbers of given numbers from the telnyx
func (h *numberHandlerTelnyx) CreateOrderNumber(customerID uuid.UUID, num string, flowID uuid.UUID, name, detail string) (*number.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"customer_id": customerID,
			"number":      num,
		},
	)

	// send a request to number providers
	numbers := []string{num}
	_, err := h.requestExternal.TelnyxNumberOrdersPost(numbers)
	if err != nil {
		log.Errorf("Could not send the order request to the telnyx. err: %v", err)
		return nil, err
	}

	// create db record for each ordered numbers
	res, err := h.createNumberByTelnyxOrderNumber(customerID, flowID, num, name, detail)
	if err != nil {
		log.Errorf("Could not handle the ordered number to the telnyx. number: %s, err: %v", num, err)
		return nil, err
	}

	return res, nil
}

// createNumberByTelnyxOrderNumber creates a number by ordered number to the telnyx.
func (h *numberHandlerTelnyx) createNumberByTelnyxOrderNumber(customerID, flowID uuid.UUID, number, name, detail string) (*number.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "createNumberByTelnyxOrderNumber",
			"customer_id":   customerID,
			"flow_id":       flowID,
			"number_number": number,
		},
	)
	ctx := context.Background()

	// get number info
	numInfos, err := h.requestExternal.TelnyxPhoneNumbersGet(1, "", number)
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
	tmpNum, err := h.requestExternal.TelnyxPhoneNumbersIDUpdateConnectionID(numInfo.ID, ConnectionID)
	if err != nil {
		log.Errorf("Could not update connection ID info. err: %v", err)
		return nil, err
	}

	tmp := tmpNum.ConvertNumber()

	// add uuid
	tmp.ID = uuid.Must(uuid.NewV4())
	tmp.CustomerID = customerID
	tmp.FlowID = flowID
	tmp.Name = name
	tmp.Detail = detail

	tmp.TMCreate = dbhandler.GetCurTime()
	tmp.TMUpdate = dbhandler.DefaultTimeStamp
	tmp.TMDelete = dbhandler.DefaultTimeStamp

	// insert into db
	if err := h.db.NumberCreate(ctx, tmp); err != nil {
		log.WithFields(
			logrus.Fields{
				"number": tmp,
			},
		).Errorf("Could not create a number. number: %s, err: %v", tmp.Number, err)
		return nil, err
	}

	// get created number
	res, err := h.db.NumberGet(ctx, tmp.ID)
	if err != nil {
		log.Errorf("Could not get created number info. err: %v", err)
		return nil, err
	}

	return res, err
}

// ReleseOrderNumbers release an existed order number from the telnyx
func (h *numberHandlerTelnyx) ReleaseOrderNumber(ctx context.Context, number *number.Number) (*number.Number, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"number": number,
		},
	)

	// delete the number from the telnyx
	phoneNumber, err := h.requestExternal.TelnyxPhoneNumbersIDDelete(number.ProviderReferenceID)
	if err != nil {
		log.Errorf("Could not delete the number from the telnyx. number: %s, err: %v", number.ID, err)
		return nil, err
	}
	log.WithFields(
		logrus.Fields{
			"phone_number": phoneNumber,
		},
	).Debugf("Release the number from the telnyx correctly. number: %s", number.ID)

	// delete from the database
	if err := h.db.NumberDelete(ctx, number.ID); err != nil {
		log.Errorf("Could not delete the number from the db. number: %s, err: %v", number.ID, err)
		return nil, err
	}

	// get deleted number
	res, err := h.db.NumberGet(ctx, number.ID)
	if err != nil {
		log.Errorf("Could not get deleted number info. number: %s, err: %v", number.ID, err)
		return nil, err
	}

	return res, nil
}
