package servicehandler

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	tkmessage "monorepo/bin-talk-manager/models/message"
)

// ServiceAgentTalkMentionList gets the talk messages that mention the calling agent, newest first.
//
// Both ids come from the authenticated identity, never from the request, so an agent can only list
// its own mentions. The access rule for the chats (deleted chats and messages, the customer, and
// the participation in direct and group chats) is applied by talk-manager in the same query.
func (h *serviceHandler) ServiceAgentTalkMentionList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*tkmessage.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentTalkMentionList",
		"customer_id": a.CustomerID,
	})

	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	messages, err := h.reqHandler.TalkV1MentionList(ctx, a.CustomerID, a.AgentID(), token, size)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get the mentions.")
	}

	res := []*tkmessage.WebhookMessage{}
	for _, m := range messages {
		wm, err := m.ConvertWebhookMessage()
		if err != nil {
			log.Errorf("Could not convert the message. message_id: %s, err: %v", m.ID, err)
			continue
		}
		res = append(res, wm)
	}

	return res, nil
}
