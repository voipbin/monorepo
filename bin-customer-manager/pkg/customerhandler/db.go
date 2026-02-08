package customerhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-customer-manager/models/customer"
)

// List returns list of customers
func (h *customerHandler) List(ctx context.Context, size uint64, token string, filters map[customer.Field]any) ([]*customer.Customer, error) {
	log := logrus.WithField("func", "List")

	res, err := h.db.CustomerList(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get customer info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Get returns customer info.
func (h *customerHandler) Get(ctx context.Context, id uuid.UUID) (*customer.Customer, error) {
	log := logrus.WithField("func", "Get")

	res, err := h.db.CustomerGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get customer info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Create creates a new customer.
func (h *customerHandler) Create(
	ctx context.Context,
	name string,
	detail string,
	email string,
	phoneNumber string,
	address string,
	webhookMethod customer.WebhookMethod,
	webhookURI string,
) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Create",
		"name":           name,
		"detail":         detail,
		"email":          email,
		"phoneNumber":    phoneNumber,
		"address":        address,
		"webhook_method": webhookMethod,
		"webhook_uri":    webhookURI,
	})
	log.Debug("Creating a new customer.")

	// check if the customer's email is already exist in the agent
	if !h.validateCreate(ctx, email) {
		log.Errorf("The email is already exist. email: %s", email)
		return nil, fmt.Errorf("the email is already exist")
	}

	id := h.utilHandler.UUIDCreate()

	// create customer
	u := &customer.Customer{
		ID: id,

		Name:   name,
		Detail: detail,

		Email:       email,
		PhoneNumber: phoneNumber,
		Address:     address,

		WebhookMethod: webhookMethod,
		WebhookURI:    webhookURI,

		EmailVerified: true,
	}

	if err := h.db.CustomerCreate(ctx, u); err != nil {
		log.Errorf("Could not create a new customer. err: %v", err)
		return nil, err
	}

	res, err := h.db.CustomerGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created customer info. err: %v", err)
		return nil, err
	}

	// notify
	h.notifyHandler.PublishEvent(ctx, customer.EventTypeCustomerCreated, res)

	return res, nil
}

// dbDelete deletes the customer.
func (h *customerHandler) dbDelete(ctx context.Context, id uuid.UUID) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "dbDelete",
		"customer_id": id,
	})
	log.Debug("Deleteing the customer.")

	if err := h.db.CustomerDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the customer. err: %v", err)
		return nil, err
	}

	// get deleted customer
	res, err := h.db.CustomerGet(ctx, id)
	if err != nil {
		// we couldn't get deleted item. but we've delete the customer already, just return here.
		log.Errorf("Could not get deleted customer. err: %v", err)
		return nil, fmt.Errorf("could not get deleted customer")
	}

	// notify
	h.notifyHandler.PublishEvent(ctx, customer.EventTypeCustomerDeleted, res)

	return res, nil
}

// UpdateBasicInfo updates the customer's basic info.
func (h *customerHandler) UpdateBasicInfo(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
	email string,
	phoneNumber string,
	address string,
	webhookMethod customer.WebhookMethod,
	webhookURI string,
) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "UpdateBasicInfo",
		"customer_id": id,
	})
	log.Debug("Updating the customer's basic info.")

	fields := map[customer.Field]any{
		customer.FieldName:          name,
		customer.FieldDetail:        detail,
		customer.FieldEmail:         email,
		customer.FieldPhoneNumber:   phoneNumber,
		customer.FieldAddress:       address,
		customer.FieldWebhookMethod: webhookMethod,
		customer.FieldWebhookURI:    webhookURI,
	}

	if err := h.db.CustomerUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update the basic info. err: %v", err)
		return nil, err
	}

	// get updated customer
	res, err := h.db.CustomerGet(ctx, id)
	if err != nil {
		// we couldn't get updated item. but we've updated the customer already, just return here.
		log.Errorf("Could not get updated customer. err: %v", err)
		return nil, fmt.Errorf("could not get updated customer")
	}

	// notify
	h.notifyHandler.PublishEvent(ctx, customer.EventTypeCustomerUpdated, res)

	return res, nil
}

// UpdateBillingAccountID updates the customer's billing accountid.
func (h *customerHandler) UpdateBillingAccountID(ctx context.Context, id uuid.UUID, billingAccountID uuid.UUID) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "UpdateBillingAccountID",
		"user_id": id,
	})
	log.Debug("Updating the customer's billing account id.")

	fields := map[customer.Field]any{
		customer.FieldBillingAccountID: billingAccountID,
	}

	if err := h.db.CustomerUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update the billing account id. err: %v", err)
		return nil, err
	}

	res, err := h.db.CustomerGet(ctx, id)
	if err != nil {
		// we couldn't get updated item. but we've updated the customer already, just return here.
		log.Errorf("Could not get updated customer. err: %v", err)
		return nil, fmt.Errorf("could not get updated customer")
	}

	// notify
	h.notifyHandler.PublishEvent(ctx, customer.EventTypeCustomerUpdated, res)

	return res, nil
}
