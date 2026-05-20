package accounthandler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
)

// Create is handy function for creating a confbridge.
// it increases corresponded counter
func (h *accountHandler) Create(ctx context.Context, customerID uuid.UUID, accountType account.Type, name string, detail string, secret string, token string, messageFlowID uuid.UUID, providerData json.RawMessage) (*account.Account, error) {
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

		Secret:        secret,
		Token:         token,
		MessageFlowID: messageFlowID,
		ProviderData:  providerData,
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
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, account.EventTypeAccountCreated, res)

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
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameConversationManager,
				"CONVERSATION_ACCOUNT_NOT_FOUND",
				"The conversation account was not found.",
			).Wrap(err)
		}
		return nil, errors.Wrap(err, "could not get account info")
	}
	log.WithField("account", res).Debugf("Retrieved account info. account_id: %s", id)

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
	log.WithField("accounts", res).Debugf("Retrieved account list. count: %d", len(res))

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

	// get the account first for teardown
	ac, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get account info for teardown. err: %v", err)
		return nil, errors.Wrap(err, "could not get account info")
	}

	// teardown external resources (best-effort)
	h.teardown(ctx, ac)

	if errDelete := h.db.AccountDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not delete account info. err: %v", errDelete)
		return nil, errors.Wrap(errDelete, "could not delete account info")
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted account info")
		return nil, errors.Wrap(err, "could not get deleted account info")
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, account.EventTypeAccountDeleted, res)

	return res, nil
}
