package customerhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
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
func (h *customerHandler) Create(ctx context.Context, username, password, name, detail string, webhookMethod customer.WebhookMethod, webhookURI string, permissionIDs []uuid.UUID) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Create",
		"username":       username,
		"name":           name,
		"detail":         detail,
		"webhook_method": webhookMethod,
		"webhook_uri":    webhookURI,
		"permission_ids": permissionIDs,
	})
	log.Debug("Creating a new customer.")

	tmp, _ := h.db.CustomerGetByUsername(ctx, username)
	if tmp != nil {
		log.Errorf("The customer is already existing. username: %s", username)
		return nil, fmt.Errorf("customer already exist")
	}

	// generate hash password
	hashPassword, err := generateHash(password)
	if err != nil {
		log.Errorf("Could not generate hash. err: %v", err)
		return nil, err
	}

	// create customer
	id := uuid.Must(uuid.NewV4())
	u := &customer.Customer{
		ID:           id,
		Username:     username,
		PasswordHash: hashPassword,

		Name:          name,
		Detail:        detail,
		WebhookMethod: webhookMethod,
		WebhookURI:    webhookURI,

		PermissionIDs: permissionIDs,

		TMCreate: dbhandler.GetCurTime(),
		TMUpdate: dbhandler.DefaultTimeStamp,
		TMDelete: dbhandler.DefaultTimeStamp,
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
	h.notifyhandler.PublishEvent(ctx, customer.EventTypeCustomerCreated, res)

	return res, nil
}

// Delete deletes the customer.
func (h *customerHandler) Delete(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Delete",
		"customer_id": id,
	})
	log.Debug("Deleteing the customer.")

	if err := h.db.CustomerDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the customer. err: %v", err)
		return err
	}

	// get deleted customer
	tmp, err := h.db.CustomerGet(ctx, id)
	if err != nil {
		// we couldn't get deleted item. but we've delete the customer already, just return here.
		log.Errorf("Could not get deleted customer. err: %v", err)
		return nil
	}

	// notify
	h.notifyhandler.PublishEvent(ctx, customer.EventTypeCustomerDeleted, tmp)

	return nil
}

// UpdateBasicInfo updates the customer's basic info.
func (h *customerHandler) UpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string, webhookMethod customer.WebhookMethod, webhookURI string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "UpdateBasicInfo",
		"customer_id": id,
	})
	log.Debug("Updating the customer's basic info.")

	if err := h.db.CustomerSetBasicInfo(ctx, id, name, detail, webhookMethod, webhookURI); err != nil {
		log.Errorf("Could not update the basic info. err: %v", err)
		return err
	}

	// get updated customer
	tmp, err := h.db.CustomerGet(ctx, id)
	if err != nil {
		// we couldn't get updated item. but we've updated the customer already, just return here.
		log.Errorf("Could not get updated customer. err: %v", err)
		return nil
	}

	// notify
	h.notifyhandler.PublishEvent(ctx, customer.EventTypeCustomerUpdated, tmp)

	return nil
}

// UpdatePassword updates the customer's password.
func (h *customerHandler) UpdatePassword(ctx context.Context, id uuid.UUID, password string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "UpdatePassword",
		"customer_id": id,
	})
	log.Debug("Updating the customer's password.")

	// generate hash password
	hashPassword, err := generateHash(password)
	if err != nil {
		log.Errorf("Could not generate hash. err: %v", err)
		return err
	}

	if err := h.db.CustomerSetPasswordHash(ctx, id, hashPassword); err != nil {
		log.Errorf("Could not update the password. err: %v", err)
		return err
	}

	tmp, err := h.db.CustomerGet(ctx, id)
	if err != nil {
		// we couldn't get updated item. but we've updated the customer already, just return here.
		log.Errorf("Could not get updated customer. err: %v", err)
		return nil
	}

	// notify
	h.notifyhandler.PublishEvent(ctx, customer.EventTypeCustomerUpdated, tmp)

	return nil
}

// UpdatePermissionIDs updates the customer's permission ids.
func (h *customerHandler) UpdatePermissionIDs(ctx context.Context, id uuid.UUID, permissionIDs []uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "UpdatePermissionIDs",
		"user_id": id,
	})
	log.Debug("Updating the customer's permission ids.")

	if err := h.db.CustomerSetPermissionIDs(ctx, id, permissionIDs); err != nil {
		log.Errorf("Could not update the permission. err: %v", err)
		return err
	}

	tmp, err := h.db.CustomerGet(ctx, id)
	if err != nil {
		// we couldn't get updated item. but we've updated the customer already, just return here.
		log.Errorf("Could not get updated customer. err: %v", err)
		return nil
	}

	// notify
	h.notifyhandler.PublishEvent(ctx, customer.EventTypeCustomerUpdated, tmp)

	return nil
}

// Login validate the customer's username and password.
func (h *customerHandler) Login(ctx context.Context, username, password string) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Login",
		"username": username,
	})
	log.Debug("Customer login.")

	res, err := h.db.CustomerGetByUsername(ctx, username)
	if err != nil {
		log.Errorf("Could not get customer info. err: %v", err)
		return nil, fmt.Errorf("no user info")
	}

	if !checkHash(password, res.PasswordHash) {
		return nil, fmt.Errorf("wrong password")
	}

	return res, nil
}
