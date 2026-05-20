package servicehandler

import (
	"context"
	"encoding/json"
	"fmt"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cvaccount "monorepo/bin-conversation-manager/models/account"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// conversationAccountGet validates the conversation account's ownership and returns the conversation account info.
func (h *serviceHandler) conversationAccountGet(ctx context.Context, conversationAccountID uuid.UUID) (*cvaccount.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                    "conversationAccountGet",
		"conversation_account_id": conversationAccountID,
	})

	// send request
	res, err := h.reqHandler.ConversationV1AccountGet(ctx, conversationAccountID)
	if err != nil {
		log.Errorf("Could not get the conversation info. err: %v", err)
		return nil, err
	}
	log.WithField("conversation", res).Debug("Received result.")

	return res, nil
}

// ConversationAccountGetsByCustomerID gets the list of conversation accounts of the given customer id.
// It returns list of conversation accounts if it succeed.
func (h *serviceHandler) ConversationAccountGetsByCustomerID(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cvaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ConversationAccountGetsByCustomerID",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a conversation accounts.")

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	filters := map[cvaccount.Field]any{
		cvaccount.FieldDeleted:    false,
		cvaccount.FieldCustomerID: a.CustomerID,
	}

	// get
	tmp, err := h.reqHandler.ConversationV1AccountList(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get conversation account infos from the conversation-manager. err: %v", err)
		return nil, fmt.Errorf("%w: could not find conversation accounts info", err)
	}

	// create result
	res := []*cvaccount.WebhookMessage{}
	for _, f := range tmp {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// ConversationAccountGet gets the conversation of the given id.
// It returns conversation account if it succeed.
func (h *serviceHandler) ConversationAccountGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cvaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                    "ConversationAccountGet",
		"customer_id":             a.CustomerID,
		"username":                a.DisplayName(),
		"conversation_account_id": id,
	})
	log.Debug("Getting an conversation account.")

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// get
	tmp, err := h.conversationAccountGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get conversation account info from the conversation-manager. err: %v", err)
		return nil, fmt.Errorf("%w: could not find conversation account info", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConversationAccountCreate creates a new conversation account
func (h *serviceHandler) ConversationAccountCreate(
	ctx context.Context,
	a *auth.AuthIdentity,
	accountType cvaccount.Type,
	name string,
	detail string,
	secret string,
	token string,
	messageFlowID uuid.UUID,
	providerData json.RawMessage,
) (*cvaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ConversationAccountCreate",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
	})
	log.Debug("Creating a new conversation account.")

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.ConversationV1AccountCreate(ctx, a.CustomerID, accountType, name, detail, secret, token, messageFlowID, providerData)
	if err != nil {
		log.Errorf("Could not create a new conversation account. err: %v", err)
		return nil, errors.Wrap(err, "could not create a new conversation account")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConversationAccountUpdate updates the conversation account
func (h *serviceHandler) ConversationAccountUpdate(ctx context.Context, a *auth.AuthIdentity, accountID uuid.UUID, fields map[cvaccount.Field]any) (*cvaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ConversationAccountUpdate",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
	})
	log.Debug("Creating a new conversation account.")

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// get campaign
	ca, err := h.conversationAccountGet(ctx, accountID)
	if err != nil {
		log.Errorf("Could not get conversation info from the conversation-manager. err: %v", err)
		return nil, fmt.Errorf("%w: could not find conversation info", err)
	}

	if !h.hasPermission(ctx, a, ca.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.ConversationV1AccountUpdate(ctx, accountID, fields)
	if err != nil {
		log.Errorf("Could not update the conversation account. err: %v", err)
		return nil, errors.Wrap(err, "could not update the conversation account")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConversationAccountDelete deletes the conversation account
func (h *serviceHandler) ConversationAccountDelete(ctx context.Context, a *auth.AuthIdentity, accountID uuid.UUID) (*cvaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ConversationAccountDelete",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
	})
	log.Debug("Creating a new conversation account.")

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// get campaign
	ca, err := h.conversationAccountGet(ctx, accountID)
	if err != nil {
		log.Errorf("Could not get conversation info from the conversation-manager. err: %v", err)
		return nil, fmt.Errorf("%w: could not find conversation info", err)
	}

	if !h.hasPermission(ctx, a, ca.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.ConversationV1AccountDelete(ctx, accountID)
	if err != nil {
		log.Errorf("Could not update the conversation account. err: %v", err)
		return nil, errors.Wrap(err, "could not update the conversation account")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
