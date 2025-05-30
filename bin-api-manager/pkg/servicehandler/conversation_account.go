package servicehandler

import (
	"context"
	"fmt"

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
func (h *serviceHandler) ConversationAccountGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cvaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ConversationAccountGetsByCustomerID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a conversation accounts.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	filters := map[cvaccount.Field]any{
		cvaccount.FieldDeleted:    false,
		cvaccount.FieldCustomerID: a.CustomerID,
	}

	// get
	tmp, err := h.reqHandler.ConversationV1AccountGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get conversation account infos from the conversation-manager. err: %v", err)
		return nil, fmt.Errorf("could not find conversation accounts info. err: %v", err)
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
func (h *serviceHandler) ConversationAccountGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cvaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                    "ConversationAccountGet",
		"customer_id":             a.CustomerID,
		"username":                a.Username,
		"conversation_account_id": id,
	})
	log.Debug("Getting an conversation account.")

	// get
	tmp, err := h.conversationAccountGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get conversation account info from the conversation-manager. err: %v", err)
		return nil, fmt.Errorf("could not find conversation account info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConversationAccountCreate creates a new conversation account
func (h *serviceHandler) ConversationAccountCreate(
	ctx context.Context,
	a *amagent.Agent,
	accountType cvaccount.Type,
	name string,
	detail string,
	secret string,
	token string,
) (*cvaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ConversationAccountCreate",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})
	log.Debug("Creating a new conversation account.")

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.ConversationV1AccountCreate(ctx, a.CustomerID, accountType, name, detail, secret, token)
	if err != nil {
		log.Errorf("Could not create a new conversation account. err: %v", err)
		return nil, errors.Wrap(err, "could not create a new conversation account")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConversationAccountUpdate updates the conversation account
func (h *serviceHandler) ConversationAccountUpdate(ctx context.Context, a *amagent.Agent, accountID uuid.UUID, fields map[cvaccount.Field]any) (*cvaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ConversationAccountUpdate",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})
	log.Debug("Creating a new conversation account.")

	// get campaign
	ca, err := h.conversationAccountGet(ctx, accountID)
	if err != nil {
		log.Errorf("Could not get conversation info from the conversation-manager. err: %v", err)
		return nil, fmt.Errorf("could not find conversation info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, ca.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
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
func (h *serviceHandler) ConversationAccountDelete(ctx context.Context, a *amagent.Agent, accountID uuid.UUID) (*cvaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ConversationAccountDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})
	log.Debug("Creating a new conversation account.")

	// get campaign
	ca, err := h.conversationAccountGet(ctx, accountID)
	if err != nil {
		log.Errorf("Could not get conversation info from the conversation-manager. err: %v", err)
		return nil, fmt.Errorf("could not find conversation info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, ca.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.ConversationV1AccountDelete(ctx, accountID)
	if err != nil {
		log.Errorf("Could not update the conversation account. err: %v", err)
		return nil, errors.Wrap(err, "could not update the conversation account")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
