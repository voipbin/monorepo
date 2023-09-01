package customerhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	"gitlab.com/voipbin/bin-manager/customer-manager.git/pkg/dbhandler"
)

// Gets returns list of customers
func (h *customerHandler) Gets(ctx context.Context, size uint64, token string) ([]*customer.Customer, error) {
	log := logrus.WithField("func", "Gets")

	res, err := h.db.CustomerGets(ctx, size, token)
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
	username string,
	password string,
	name string,
	detail string,
	email string,
	phoneNumber string,
	address string,
	webhookMethod customer.WebhookMethod,
	webhookURI string,
	permissionIDs []uuid.UUID,
) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Create",
		"username":       username,
		"name":           name,
		"detail":         detail,
		"email":          email,
		"phoneNumber":    phoneNumber,
		"address":        address,
		"webhook_method": webhookMethod,
		"webhook_uri":    webhookURI,
		"permission_ids": permissionIDs,
	})
	log.Debug("Creating a new customer.")

	// verify the username is unique
	tmp, _ := h.db.CustomerGetByUsername(ctx, username)
	if tmp != nil {
		log.Errorf("The customer is already existing. username: %s", username)
		return nil, fmt.Errorf("customer already exist")
	}

	id := h.utilHandler.UUIDCreate()

	// create billingAccount
	billingAccount, err := h.reqHandler.BillingV1AccountCreate(ctx, id, "basic billing account", "billing account for default use")
	if err != nil {
		log.Errorf("Could not create a billing account info. err: %v", err)
		return nil, errors.Wrap(err, "could not create a billing account info")
	}
	log.WithField("billing_account", billingAccount).Debugf("Created a billing account for new customer. customer_id: %s", id)

	// generate hash password
	hashPassword, err := h.helpHandler.HashGenerate(password)
	if err != nil {
		log.Errorf("Could not generate hash. err: %v", err)
		return nil, err
	}

	// create customer
	u := &customer.Customer{
		ID:           id,
		Username:     username,
		PasswordHash: hashPassword,

		Name:   name,
		Detail: detail,

		Email:       email,
		PhoneNumber: phoneNumber,
		Address:     address,

		WebhookMethod: webhookMethod,
		WebhookURI:    webhookURI,

		PermissionIDs:    permissionIDs,
		BillingAccountID: billingAccount.ID,
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

// Delete deletes the customer.
func (h *customerHandler) Delete(ctx context.Context, id uuid.UUID) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Delete",
		"customer_id": id,
	})
	log.Debug("Deleteing the customer.")

	// get billing accounts
	as, err := h.reqHandler.BillingV1AccountGets(ctx, id, "", 100)
	if err != nil {
		log.Errorf("Could not get customer's billing accounts. err: %v", err)
		return nil, errors.Wrap(err, "could not get customer's billing accounts")
	}

	for _, ac := range as {
		if ac.TMDelete < dbhandler.DefaultTimeStamp {
			// already deleted
			continue
		}
		_, err := h.reqHandler.BillingV1AccountDelete(ctx, ac.ID)
		if err != nil {
			log.Errorf("Could not delete the billing account. err: %v", err)
			// we've got an error here, but keep moving.
		}
	}

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

	if err := h.db.CustomerSetBasicInfo(
		ctx,
		id,
		name,
		detail,
		email,
		phoneNumber,
		address,
		webhookMethod,
		webhookURI,
	); err != nil {
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

// UpdatePassword updates the customer's password.
func (h *customerHandler) UpdatePassword(ctx context.Context, id uuid.UUID, password string) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "UpdatePassword",
		"customer_id": id,
	})
	log.Debug("Updating the customer's password.")

	// generate hash password
	hashPassword, err := h.helpHandler.HashGenerate(password)
	if err != nil {
		log.Errorf("Could not generate hash. err: %v", err)
		return nil, err
	}

	if err := h.db.CustomerSetPasswordHash(ctx, id, hashPassword); err != nil {
		log.Errorf("Could not update the password. err: %v", err)
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

// UpdatePermissionIDs updates the customer's permission ids.
func (h *customerHandler) UpdatePermissionIDs(ctx context.Context, id uuid.UUID, permissionIDs []uuid.UUID) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "UpdatePermissionIDs",
		"user_id": id,
	})
	log.Debug("Updating the customer's permission ids.")

	if err := h.db.CustomerSetPermissionIDs(ctx, id, permissionIDs); err != nil {
		log.Errorf("Could not update the permission. err: %v", err)
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

// UpdateBillingAccountID updates the customer's billing accountid.
func (h *customerHandler) UpdateBillingAccountID(ctx context.Context, id uuid.UUID, billingAccountID uuid.UUID) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "UpdateBillingAccountID",
		"user_id": id,
	})
	log.Debug("Updating the customer's billing account id.")

	if err := h.db.CustomerSetBillingAccountID(ctx, id, billingAccountID); err != nil {
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
