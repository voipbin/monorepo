package servicehandler

import (
	"context"
	"fmt"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// conversationGet validates the conversation's ownership and returns the conversation info.
func (h *serviceHandler) conversationGet(ctx context.Context, conversationID uuid.UUID) (*cvconversation.Conversation, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conversationGet",
		"conversation_id": conversationID,
	})

	// send request
	res, err := h.reqHandler.ConversationV1ConversationGet(ctx, conversationID)
	if err != nil {
		log.Errorf("Could not get the conversation info. err: %v", err)
		return nil, err
	}
	log.WithField("conversation", res).Debug("Received result.")

	return res, nil
}

// ConversationGetsByCustomerID gets the list of conversations of the given customer id.
// It returns list of conversations if it succeed.
func (h *serviceHandler) ConversationGetsByCustomerID(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cvconversation.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ConversationGetsByCustomerID",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a conversations.")

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

	filters := map[cvconversation.Field]any{
		cvconversation.FieldDeleted:    false,
		cvconversation.FieldCustomerID: a.CustomerID,
	}

	tmps, err := h.conversationList(ctx, a, size, token, filters)
	if err != nil {
		log.Errorf("Could not get conversations. err: %v", err)
		return nil, errors.Wrapf(err, "Could not get conversations.")
	}

	// create result
	res := []*cvconversation.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// conversationGets gets the list of conversations.
// It returns list of conversations if it succeed.
func (h *serviceHandler) conversationList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, fields map[cvconversation.Field]any) ([]cvconversation.Conversation, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "ConversationGetsByCustomerID",
		"agent": a,
		"size":  size,
		"token": token,
	})
	log.Debug("Getting a conversations.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// gets
	res, err := h.reqHandler.ConversationV1ConversationList(ctx, token, size, fields)
	if err != nil {
		log.Errorf("Could not get campaigns info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("%w: could not find campaigns info", err)
	}

	return res, nil
}

// ConversationGet gets the conversation of the given id.
// It returns conversation if it succeed.
func (h *serviceHandler) ConversationGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cvconversation.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ConversationGet",
		"customer_id":     a.CustomerID,
		"username":        a.DisplayName(),
		"conversation_id": id,
	})
	log.Debug("Getting an conversation.")

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// get campaign
	tmp, err := h.conversationGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get conversation info from the conversation-manager. err: %v", err)
		return nil, fmt.Errorf("%w: could not find conversation info", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConversationUpdate update the conversation of the given id.
// It returns updated conversation if it succeed.
func (h *serviceHandler) ConversationUpdate(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID, fields map[cvconversation.Field]any) (*cvconversation.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ConversationUpdate",
		"customer_id":     a.CustomerID,
		"username":        a.DisplayName(),
		"conversation_id": conversationID,
	})
	log.Debug("Updating the conversation.")

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// get campaign
	c, err := h.conversationGet(ctx, conversationID)
	if err != nil {
		log.Errorf("Could not get conversation info from the conversation-manager. err: %v", err)
		return nil, fmt.Errorf("%w: could not find conversation info", err)
	}

	// admin/manager retain unrestricted access to the existing forward path. For non-admin
	// callers the only allowed update is the owning-agent self-unassign (see design §5.2).
	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		if !a.IsAgent() || a.Agent == nil {
			log.Info("The agent has no permission for this agent.")
			return nil, serviceerrors.ErrPermissionDenied
		}
		if c.OwnerType != commonidentity.OwnerTypeAgent || c.OwnerID != a.Agent.ID {
			log.Info("The agent has no permission for this agent.")
			return nil, serviceerrors.ErrPermissionDenied
		}
		if !payloadIsExactlySelfUnassign(fields) {
			log.Info("The agent has no permission for this agent.")
			return nil, serviceerrors.ErrPermissionDenied
		}
	}

	tmp, err := h.reqHandler.ConversationV1ConversationUpdate(ctx, conversationID, fields)
	if err != nil {
		log.Errorf("Could not update the conversation. err: %v", err)
		return nil, errors.Wrap(err, "could not update the conversation")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// payloadIsExactlySelfUnassign returns true iff the partial-update fields map represents
// exactly a self-unassign: a single key (FieldOwnerID) whose value is uuid.Nil.
// FieldOwnerType MUST NOT be present, even if redundant given len==1 — the explicit
// check closes the OpenAPI-bypass attack where an agent caller sends {owner_id, owner_type}
// together. See design §5.2.
func payloadIsExactlySelfUnassign(fields map[cvconversation.Field]any) bool {
	if len(fields) != 1 {
		return false
	}
	v, ok := fields[cvconversation.FieldOwnerID]
	if !ok {
		return false
	}
	ownerID, okType := v.(uuid.UUID)
	if !okType || ownerID != uuid.Nil {
		return false
	}
	if _, hasOwnerType := fields[cvconversation.FieldOwnerType]; hasOwnerType {
		return false
	}
	return true
}
