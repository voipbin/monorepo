package servicehandler

import (
	"context"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// ServiceAgentConversationGet gets the conversation of the given id.
// It returns conversation if it succeed.
func (h *serviceHandler) ServiceAgentConversationGet(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID) (*cvconversation.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	// get
	tmp, err := h.conversationGet(ctx, conversationID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get conversation.")
	}

	if tmp.OwnerID != a.AgentID() {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentConversationGets sends a request to conversation-manager
// to getting the list of conversation.
// it returns list of chatroom messages if it succeed.
func (h *serviceHandler) ServiceAgentConversationList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cvconversation.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// filters
	filters := map[cvconversation.Field]any{
		cvconversation.FieldDeleted: false,
		cvconversation.FieldOwnerID: a.AgentID(),
	}

	tmps, err := h.conversationList(ctx, a, size, token, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get the conversations.")
	}

	// create result
	res := []*cvconversation.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// ServiceAgentConversationUpdate updates the conversation of the given id for a service-agent caller.
// Only admin/manager callers are permitted.
func (h *serviceHandler) ServiceAgentConversationUpdate(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID, fields map[cvconversation.Field]any) (*cvconversation.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	// get
	c, err := h.conversationGet(ctx, conversationID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get conversation.")
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	updated, err := h.reqHandler.ConversationV1ConversationUpdate(ctx, conversationID, fields)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not update conversation.")
	}

	return updated.ConvertWebhookMessage(), nil
}

// ServiceAgentConversationUnassign removes the agent as the owner of the given conversation.
// Admin/manager callers may unassign any conversation. The owning agent may self-unassign.
func (h *serviceHandler) ServiceAgentConversationUnassign(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID) (*cvconversation.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	// get
	c, err := h.conversationGet(ctx, conversationID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get conversation.")
	}

	isAdminOrManager := h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager)
	isOwningAgent := a.Agent != nil &&
		c.OwnerType == commonidentity.OwnerTypeAgent &&
		c.OwnerID == a.Agent.ID

	if !isAdminOrManager && !isOwningAgent {
		return nil, serviceerrors.ErrPermissionDenied
	}

	unassignFields := map[cvconversation.Field]any{
		cvconversation.FieldOwnerID: uuid.Nil,
	}

	updated, err := h.reqHandler.ConversationV1ConversationUpdate(ctx, conversationID, unassignFields)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not unassign conversation.")
	}

	return updated.ConvertWebhookMessage(), nil
}
