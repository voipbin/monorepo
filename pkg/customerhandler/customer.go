package customerhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	"gitlab.com/voipbin/bin-manager/customer-manager.git/pkg/dbhandler"
)

// CustomerGets returns list of customers
func (h *customerHandler) CustomerGets(ctx context.Context, size uint64, token string) ([]*customer.Customer, error) {
	log := logrus.WithField("func", "CustomerGets")

	res, err := h.db.CustomerGets(ctx, size, token)
	if err != nil {
		log.Errorf("Could not get customer info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// CustomerGet returns customer info.
func (h *customerHandler) CustomerGet(ctx context.Context, id uuid.UUID) (*customer.Customer, error) {
	log := logrus.WithField("func", "CustomerGet")

	res, err := h.db.CustomerGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get customer info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// CustomerCreate creates a new customer.
func (h *customerHandler) CustomerCreate(ctx context.Context, username, password, name, detail, webhookMethod, webhookURI string, permissionIDs []uuid.UUID) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "CustomerCreate",
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

	return res, nil
}

// CustomerDelete deletes the customer.
func (h *customerHandler) CustomerDelete(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerDelete",
		"customer_id": id,
	})
	log.Debug("Deleteing the customer.")

	if err := h.db.CustomerDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the customer. err: %v", err)
		return err
	}

	return nil
}

// CustomerUpdateBasicInfo updates the customer's basic info.
func (h *customerHandler) CustomerUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail, webhookMethod, webhookURI string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerUpdateBasicInfo",
		"customer_id": id,
	})
	log.Debug("Updating the customer's basic info.")

	if err := h.db.CustomerSetBasicInfo(ctx, id, name, detail, webhookMethod, webhookURI); err != nil {
		log.Errorf("Could not update the basic info. err: %v", err)
		return err
	}

	return nil
}

// CustomerUpdatePassword updates the customer's password.
func (h *customerHandler) CustomerUpdatePassword(ctx context.Context, id uuid.UUID, password string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerUpdatePassword",
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

	return nil
}

// CustomerUpdatePermissionIDs updates the customer's permission ids.
func (h *customerHandler) CustomerUpdatePermissionIDs(ctx context.Context, id uuid.UUID, permissionIDs []uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "CustomerUpdatePermissionIDs",
		"user_id": id,
	})
	log.Debug("Updating the customer's permission ids.")

	if err := h.db.CustomerSetPermissionIDs(ctx, id, permissionIDs); err != nil {
		log.Errorf("Could not update the permission. err: %v", err)
		return err
	}

	return nil
}

// CustomerLogin validate the customer's username and password.
func (h *customerHandler) CustomerLogin(ctx context.Context, username, password string) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "CustomerLogin",
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
