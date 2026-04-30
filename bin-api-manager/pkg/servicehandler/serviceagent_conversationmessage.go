package servicehandler

import (
	"context"
	"fmt"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cvmedia "monorepo/bin-conversation-manager/models/media"
	cvmessage "monorepo/bin-conversation-manager/models/message"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ServiceAgentConversationMessageGets sends a request to conversation-manager
// to getting the list of conversation messages of the given conversation id.
// it returns list of conversation messages if it succeed.
func (h *serviceHandler) ServiceAgentConversationMessageList(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID, size uint64, token string) ([]*cvmessage.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	cv, err := h.conversationGet(ctx, conversationID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get conversation.")
	}

	isAdminOrManager := h.hasPermission(ctx, a, cv.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager)
	if !isAdminOrManager && cv.OwnerID != a.AgentID() {
		return nil, serviceerrors.ErrPermissionDenied
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	tmps, err := h.conversationMessageGetsByConversationID(ctx, a, conversationID, size, token)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get conversation messages.")
	}

	// create result
	res := []*cvmessage.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// ServiceAgentConversationMessageSend send a message to the conversation.
func (h *serviceHandler) ServiceAgentConversationMessageSend(
	ctx context.Context,
	a *auth.AuthIdentity,
	conversationID uuid.UUID,
	text string,
	medias []cvmedia.Media,
) (*cvmessage.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":            "ServiceAgentConversationMessageSend",
		"customer_id":     a.CustomerID,
		"conversation_id": conversationID,
	})
	log.Debugf("Sending a message. conversation_id: %s", conversationID)

	// get conversation to check the permission
	c, err := h.conversationGet(ctx, conversationID)
	if err != nil {
		log.Errorf("Could not get conversation info. err: %v", err)
		return nil, fmt.Errorf("%w: could not verify the conversation", err)
	}

	isAdminOrManager := h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager)
	if !isAdminOrManager && c.OwnerID != a.AgentID() {
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.conversationMessageSend(ctx, a, conversationID, text, medias)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not send the message.")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
