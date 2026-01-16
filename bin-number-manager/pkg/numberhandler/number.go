package numberhandler

import (
	"context"
	"fmt"

	bmbilling "monorepo/bin-billing-manager/models/billing"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-number-manager/models/number"
)

// Create creates a new order numbers of given numbers
func (h *numberHandler) Create(ctx context.Context, customerID uuid.UUID, num string, callFlowID uuid.UUID, messageFlowID uuid.UUID, name string, detail string) (*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Create",
		"customer_id":   customerID,
		"flow_id":       callFlowID,
		"target_number": num,
	})
	log.Debugf("Creating a new number. customer_id: %s, number: %v", customerID, num)

	// check the customer has enough balance
	valid, err := h.reqHandler.CustomerV1CustomerIsValidBalance(ctx, customerID, bmbilling.ReferenceTypeNumber, "", 1)
	if err != nil {
		log.Errorf("Could not validate the customer's balance. err: %v", err)
		return nil, errors.Wrap(err, "could not validate the customer's balance")
	}

	if !valid {
		log.Errorf("The customer has not enough balance. valid: %v", valid)
		return nil, fmt.Errorf("the customer has not enough balance")
	}

	// use telnyx as a default
	tmp, err := h.numberHandlerTelnyx.NumberPurchase(num)
	if err != nil {
		log.Errorf("Could not create a number from the telnyx. err: %v", err)
		return nil, fmt.Errorf("could not create a number from the telnyx. err: %v", err)
	}

	res, err := h.Register(
		ctx,
		customerID,
		num,
		callFlowID,
		messageFlowID,
		name,
		detail,
		number.ProviderNameTelnyx,
		tmp.ID,
		tmp.Status,
		tmp.T38Enabled,
		tmp.EmergencyEnabled,
	)
	if err != nil {
		log.Errorf("Could not create the number record. err: %v", err)
		return nil, errors.Wrap(err, "could not create the number record")
	}
	log.WithField("number", res).Debugf("Created number correctly. number_id: %s", res.ID)

	// generate and update purchased number's tags
	tags := h.generateTags(ctx, res)
	if errUpdate := h.numberHandlerTelnyx.NumberUpdateTags(ctx, res, tags); errUpdate != nil {
		log.Errorf("Could not update the number tags from the provider. err: %v", errUpdate)
	}

	return res, nil
}

// Register adds a number record to the database without purchasing it from a provider.
// Unlike Create, which purchases the number from a provider (e.g. Telnyx), Register is used for existing numbers.
func (h *numberHandler) Register(
	ctx context.Context,
	customerID uuid.UUID,
	num string,
	callFlowID uuid.UUID,
	messageFlowID uuid.UUID,
	name string,
	detail string,
	providerName number.ProviderName,
	providerReferenceID string,
	status number.Status,
	t38Enabled bool,
	emergencyEnabled bool,
) (*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "Register",
		"number": num,
	})
	log.Debugf("Registering number. number: %s", num)

	filters := map[number.Field]any{
		number.FieldDeleted: false,
		number.FieldNumber:  num,
	}

	existedNumbers, err := h.dbList(ctx, 1, "", filters)
	if err != nil {
		log.Errorf("Could not check existing number. number: %s, err: %v", num, err)
		return nil, errors.Wrap(err, "could not check existing number")
	}
	if len(existedNumbers) > 0 {
		log.Errorf("The number already exists. number: %s", num)
		return nil, fmt.Errorf("the number already exists")
	}

	res, err := h.dbCreate(
		ctx,
		customerID,
		num,
		callFlowID,
		messageFlowID,
		name,
		detail,
		providerName,
		providerReferenceID,
		status,
		t38Enabled,
		emergencyEnabled,
	)
	if err != nil {
		log.Errorf("Could not create the number record. err: %v", err)
		return nil, errors.Wrap(err, "could not create the number record")
	}
	log.WithField("number", res).Debugf("Registered number correctly. number_id: %s", res.ID)

	return res, nil
}

// Delete release/deleted an existed ordered number
func (h *numberHandler) Delete(ctx context.Context, id uuid.UUID) (*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "Release",
		"number_id": id,
	})
	log.Debugf("Deleting the number. number_id: %s", id)

	num, err := h.Get(ctx, id)
	if err != nil {
		logrus.Errorf("Could not get order number info. number: %s, err: %v", id, err)
		return nil, err
	}
	log.Debugf("Deleting number info. number: %s", num.Number)

	// send delete request by provider
	switch num.ProviderName {
	case number.ProviderNameTelnyx:
		err = h.numberHandlerTelnyx.NumberRelease(ctx, num)

	case number.ProviderNameTwilio:
		err = h.numberHandlerTwilio.ReleaseNumber(ctx, num)

	default:
		err = fmt.Errorf("unsupported number provider. provider_name: %s", num.ProviderName)
	}

	if err != nil {
		log.Errorf("Could not release the number. err: %v", err)
		return nil, errors.Wrap(err, "could not release the number")
	}

	res, err := h.dbDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete number. err: %v", err)
		return nil, errors.Wrap(err, "could not delete number")
	}

	return res, nil
}

// Get returns number info of the given id
func (h *numberHandler) Get(ctx context.Context, id uuid.UUID) (*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "Get",
		"number_id": id,
	})

	res, err := h.dbGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get number info. err: %v", err)
		return nil, errors.Wrap(err, "could not get number info")
	}

	return res, nil
}

// List returns list of numbers info of the given filters
func (h *numberHandler) List(ctx context.Context, pageSize uint64, pageToken string, filters map[number.Field]any) ([]*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "List",
		"page_size":  pageSize,
		"page_token": pageToken,
		"filters":    filters,
	})
	log.Debugf("List.")

	res, err := h.dbList(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get numbers. err: %v", err)
		return nil, errors.Wrap(err, "could not get numbers")
	}

	return res, nil
}

// Update updates the number with the given fields.
func (h *numberHandler) Update(ctx context.Context, id uuid.UUID, fields map[number.Field]any) (*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "Update",
		"number_id": id,
		"fields":    fields,
	})
	log.Debugf("Update. number_id: %s", id)

	res, err := h.dbUpdate(ctx, id, fields, number.EventTypeNumberUpdated)
	if err != nil {
		log.Errorf("Could not update the number. err: %v", err)
		return nil, errors.Wrap(err, "could not update the number")
	}

	return res, nil
}
