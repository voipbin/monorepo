package numberhandler

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"

	bmaccount "monorepo/bin-billing-manager/models/account"
	bmbilling "monorepo/bin-billing-manager/models/billing"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/provider"
	"monorepo/bin-number-manager/models/providernumber"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/providerhandler"
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

	// reject reserved virtual number range for normal number creation
	if strings.HasPrefix(num, number.VirtualNumberReservedPrefix) {
		return nil, fmt.Errorf("numbers starting with %s are reserved", number.VirtualNumberReservedPrefix)
	}

	// check the customer has enough balance
	valid, err := h.reqHandler.BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, bmbilling.ReferenceTypeNumber, "", 1)
	if err != nil {
		log.Errorf("Could not validate the customer's balance. err: %v", err)
		return nil, errors.Wrap(err, "could not validate the customer's balance")
	}

	if !valid {
		log.Errorf("The customer has insufficient balance. valid: %v", valid)
		return nil, fmt.Errorf("insufficient balance for number")
	}

	// the number must not be registered yet, or the purchase would leak when the registration fails
	existed, err := h.dbList(ctx, 1, "", map[number.Field]any{
		number.FieldDeleted: false,
		number.FieldNumber:  num,
	})
	if err != nil {
		log.Errorf("Could not check the existing number. err: %v", err)
		return nil, errors.Wrap(err, "could not check existing number")
	}
	if len(existed) > 0 {
		log.Errorf("The number already exists.")
		return nil, fmt.Errorf("the number already exists")
	}

	bought, tmp, err := h.purchase(ctx, num)
	if err != nil {
		return nil, err
	}

	res, err := h.Register(
		ctx,
		customerID,
		num,
		callFlowID,
		messageFlowID,
		name,
		detail,
		number.TypeNormal,
		bought.Name,
		tmp.ID,
		tmp.Status,
		tmp.T38Enabled,
		tmp.EmergencyEnabled,
	)
	if err != nil {
		log.Errorf("Could not create the number record. err: %v", err)

		// rollback: release the purchased number from the provider it was bought from
		releaseNum := &number.Number{
			ProviderName:        bought.Name,
			ProviderReferenceID: tmp.ID,
		}
		if errRelease := bought.Provider.NumberRelease(ctx, releaseNum); errRelease != nil {
			log.Errorf("ROLLBACK_RELEASE_FAILED. Could not rollback purchased number from provider. provider_name: %s, provider_reference_id: %s, err: %v", bought.Name, tmp.ID, errRelease)
		}

		return nil, errors.Wrap(err, "could not create the number record")
	}
	log.WithField("number", res).Debugf("Created number correctly. number_id: %s", res.ID)

	// generate and update purchased number's tags
	tags := h.generateTags(ctx, res)
	if errUpdate := bought.Provider.NumberUpdateTags(ctx, res, tags); errUpdate != nil {
		log.Errorf("Could not update the number tags from the provider. err: %v", errUpdate)
	}

	return res, nil
}

// CreateVirtual creates a virtual number without provider purchase or billing.
func (h *numberHandler) CreateVirtual(ctx context.Context, customerID uuid.UUID, num string, callFlowID uuid.UUID, messageFlowID uuid.UUID, name string, detail string, allowReserved bool) (*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "CreateVirtual",
		"customer_id":   customerID,
		"target_number": num,
	})
	log.Debugf("Creating a virtual number. customer_id: %s, number: %v", customerID, num)

	// validate virtual number format
	if err := number.ValidateVirtualNumber(num, allowReserved); err != nil {
		log.Errorf("Invalid virtual number format. err: %v", err)
		return nil, fmt.Errorf("invalid virtual number format: %w", err)
	}

	// check resource limit
	valid, err := h.reqHandler.BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeVirtualNumber)
	if err != nil {
		log.Errorf("Could not validate resource limit. err: %v", err)
		return nil, fmt.Errorf("could not validate resource limit: %w", err)
	}
	if !valid {
		log.Infof("Resource limit exceeded for customer. customer_id: %s", customerID)
		return nil, fmt.Errorf("resource limit exceeded")
	}

	// register without provider
	res, err := h.Register(
		ctx,
		customerID,
		num,
		callFlowID,
		messageFlowID,
		name,
		detail,
		number.TypeVirtual,
		number.ProviderNameNone,
		"",
		number.StatusActive,
		false,
		false,
	)
	if err != nil {
		log.Errorf("Could not create virtual number. err: %v", err)
		return nil, errors.Wrap(err, "could not create virtual number")
	}
	log.WithField("number", res).Debugf("Created virtual number. number_id: %s", res.ID)

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
	numType number.Type,
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
		numType,
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
	if num.ProviderName == number.ProviderNameNone {
		// virtual number or no provider — skip provider release
		log.Debugf("No provider to release for number. number: %s", num.Number)
	} else {
		err = h.releaseByProvider(ctx, num)
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
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameNumberManager,
				"NUMBER_NOT_FOUND",
				"The phone number was not found.",
			).Wrap(err)
		}
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

// CountVirtualByCustomerID returns the count of active virtual numbers for a customer.
func (h *numberHandler) CountVirtualByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CountVirtualByCustomerID",
		"customer_id": customerID,
	})

	count, err := h.db.NumberCountVirtualByCustomerID(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get virtual number count. err: %v", err)
		return 0, errors.Wrap(err, "could not get virtual number count")
	}
	log.Debugf("Virtual number count. customer_id: %s, count: %d", customerID, count)

	return count, nil
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

// UpdateMetadata updates the number's metadata.
func (h *numberHandler) UpdateMetadata(ctx context.Context, id uuid.UUID, metadata number.Metadata) (*number.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "UpdateMetadata",
		"number_id": id,
		"metadata":  metadata,
	})
	log.Debugf("UpdateMetadata. number_id: %s", id)

	fields := map[number.Field]any{
		number.FieldMetadata: metadata,
	}

	res, err := h.dbUpdate(ctx, id, fields, number.EventTypeNumberUpdated)
	if err != nil {
		log.Errorf("Could not update the number metadata. err: %v", err)
		return nil, errors.Wrap(err, "could not update the number metadata")
	}

	return res, nil
}

// purchase buys the number from the usable providers in priority order. It falls back to the
// next provider only when the provider definitely refused the purchase. Any other failure may
// have created an order, so the purchase stops instead of risking a double purchase.
func (h *numberHandler) purchase(ctx context.Context, num string) (*providerhandler.Candidate, *providernumber.ProviderNumber, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "purchase",
		"target_number": num,
	})

	candidates, err := h.providerHandler.PurchaseCandidates(ctx)
	if err != nil {
		log.Errorf("Could not get the purchase candidates. err: %v", err)
		return nil, nil, errors.Wrap(err, "could not get the purchase candidates")
	}
	if len(candidates) == 0 {
		log.Errorf("There is no usable provider.")
		return nil, nil, fmt.Errorf("could not purchase: no usable provider")
	}

	var lastErr error
	for _, c := range candidates {
		if errCtx := ctx.Err(); errCtx != nil {
			return nil, nil, errors.Wrap(errCtx, "the purchase is canceled")
		}

		tmp, errPurchase := c.Provider.NumberPurchase(num)
		if errPurchase == nil {
			providerhandler.ObservePurchase(c.Name, "success")
			return c, tmp, nil
		}

		if stderrors.Is(errPurchase, provider.ErrPurchaseRejected) {
			providerhandler.ObservePurchase(c.Name, "rejected")
			log.Errorf("The provider rejected the purchase. Trying the next provider. provider_name: %s, err: %v", c.Name, errPurchase)
			lastErr = errPurchase
			continue
		}

		providerhandler.ObservePurchase(c.Name, "unconfirmed")
		log.Errorf("PURCHASE_UNCONFIRMED. The purchase result is unknown, stopping without a fallback. provider_name: %s, err: %v", c.Name, errPurchase)
		if stderrors.Is(errPurchase, provider.ErrPurchaseUnconfirmed) {
			return nil, nil, fmt.Errorf("could not purchase the number: %w", errPurchase)
		}
		return nil, nil, fmt.Errorf("could not purchase the number: %w: %w", provider.ErrPurchaseUnconfirmed, errPurchase)
	}

	return nil, nil, fmt.Errorf("could not purchase the number from any provider: %w", lastErr)
}

// releaseByProvider releases the number from the provider the number was bought from. The
// enabled flag of the provider is not checked, so the numbers of a disabled provider can be released.
func (h *numberHandler) releaseByProvider(ctx context.Context, num *number.Number) error {
	p, err := h.providerHandler.ForRelease(ctx, num.ProviderName)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return fmt.Errorf("unsupported number provider. provider_name: %s", num.ProviderName)
		}
		return err
	}

	return p.NumberRelease(ctx, num)
}
