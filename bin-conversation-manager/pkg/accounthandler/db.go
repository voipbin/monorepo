package accounthandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-conversation-manager/models/account"
)

// Create is handy function for creating a confbridge.
// it increases corresponded counter
func (h *accountHandler) Create(ctx context.Context, customerID uuid.UUID, accountType account.Type, name string, detail string, secret string, token string) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
		"type":        accountType,
		"name":        name,
		"detail":      detail,
		"secret":      len(secret),
		"token":       len(token),
	})

	id := h.utilHandler.UUIDCreate()
	ac := &account.Account{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},

		Type: accountType,

		Name:   name,
		Detail: detail,

		Secret: secret,
		Token:  token,
	}

	// setup the account
	if errSetup := h.setup(ctx, ac); errSetup != nil {
		log.Errorf("Could not setup the account. err: %v", errSetup)
		return nil, errors.Wrap(errSetup, "could not setup the account")
	}

	// create a account
	if errCreate := h.db.AccountCreate(ctx, ac); errCreate != nil {
		return nil, fmt.Errorf("could not create a conference. err: %v", errCreate)
	}
	promAccountCreateTotal.WithLabelValues(string(accountType)).Inc()

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get created confbridge info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, account.EventTypeAccountCreated, res)

	return res, nil
}

// Get returns an account info
func (h *accountHandler) Get(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Get",
		"account_id": id,
	})

	res, err := h.db.AccountGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get account info. err: %v", err)
		return nil, errors.Wrap(err, "could not get account info")
	}

	return res, nil
}

// List returns list of accounts of the given filters
func (h *accountHandler) List(ctx context.Context, pageToken string, pageSize uint64, filters map[account.Field]any) ([]*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "List",
		"filters": filters,
	})

	res, err := h.db.AccountList(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get messages. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Update updates account and return a updated account.
func (h *accountHandler) Update(ctx context.Context, id uuid.UUID, fields map[account.Field]any) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "Update",
		"id":     id,
		"fields": fields,
	})
	log.Debugf("Updating account. account_id: %s", id)

	if errUpdate := h.db.AccountUpdate(ctx, id, fields); errUpdate != nil {
		return nil, errors.Wrapf(errUpdate, "Could not update account. err: %v", errUpdate)
	}

	res, err := h.db.AccountGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get updated account. err: %v", err)
	}

	if errSetup := h.setup(ctx, res); errSetup != nil {
		log.Errorf("Could not setup the account. err: %v", errSetup)
		return nil, errors.Wrap(errSetup, "could not setup the account")
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, account.EventTypeAccountUpdated, res)

	return res, nil
}

// Delete deletes the account and return the deleted account
func (h *accountHandler) Delete(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Delete",
		"account_id": id,
	})

	if errDelete := h.db.AccountDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not delete account info. err: %v", errDelete)
		return nil, errors.Wrap(errDelete, "could not delete account info")
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted account info")
		return nil, errors.Wrap(err, "could not get deleted account info")
	}
	h.notifyHandler.PublishEvent(ctx, account.EventTypeAccountDeleted, res)

	return res, nil
}
