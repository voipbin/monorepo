package servicehandler

import (
	"context"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"

	amagent "monorepo/bin-agent-manager/models/agent"

	wcmessage "monorepo/bin-webchat-manager/models/message"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// webchatMessageGet validates the message's ownership and returns the message info.
func (h *serviceHandler) webchatMessageGet(ctx context.Context, id uuid.UUID) (*wcmessage.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "webchatMessageGet",
		"message_id": id,
	})

	res, err := h.reqHandler.WebchatV1MessageGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the message info. err: %v", err)
		return nil, err
	}
	log.WithField("message", res).Debug("Received result.")

	return res, nil
}

// WebchatMessageGet sends a request to webchat-manager to get the message.
func (h *serviceHandler) WebchatMessageGet(ctx context.Context, a *auth.AuthIdentity, messageID uuid.UUID) (*wcmessage.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "WebchatMessageGet",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"message_id":  messageID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	tmp, err := h.webchatMessageGet(ctx, messageID)
	if err != nil {
		log.Errorf("Could not validate the message info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// WebchatMessageList sends a request to webchat-manager to get a list of messages.
func (h *serviceHandler) WebchatMessageList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*wcmessage.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "WebchatMessageList",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	filters := map[wcmessage.Field]any{
		wcmessage.FieldCustomerID: a.CustomerID,
		wcmessage.FieldDeleted:    false,
	}

	tmps, err := h.reqHandler.WebchatV1MessageList(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get messages from the webchat-manager. err: %v", err)
		return nil, err
	}

	res := []*wcmessage.WebhookMessage{}
	for _, tmp := range tmps {
		e := tmp.ConvertWebhookMessage()
		res = append(res, e)
	}

	return res, nil
}

// WebchatMessageCreate sends a request to webchat-manager to create (send) a
// message on a session. Reachable by both the widget's direct-scope JWT
// (visitor-authored inbound messages) and an authenticated agent/accesskey
// (agent-authored outbound replies), mirroring aicall.go's dual-path auth.
// senderID is uuid.Nil for the direct-token (visitor) path.
//
// requestedDirection is intentionally NOT trusted: the actual message
// direction is derived from which auth path authenticated the caller (see
// below), never from the caller-supplied value, to prevent
// message-direction spoofing (a visitor forging direction=outbound to
// appear as an agent reply, or an agent forging direction=inbound).
func (h *serviceHandler) WebchatMessageCreate(
	ctx context.Context,
	a *auth.AuthIdentity,
	sessionID uuid.UUID,
	requestedDirection wcmessage.Direction,
	text string,
) (*wcmessage.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "WebchatMessageCreate",
		"customer_id": a.CustomerID,
		"session_id":  sessionID,
	})
	_ = requestedDirection // never trusted; direction is derived below per auth path

	senderID := uuid.Nil
	var direction wcmessage.Direction

	switch {
	case a.IsAgent() || a.IsAccesskey():
		// Resolve the target session first and check permission against
		// the SESSION's actual owner (s.CustomerID), not the caller's own
		// a.CustomerID -- the latter is a tautology that any customer
		// admin/manager trivially passes regardless of which customer's
		// session they're targeting, mirroring WebchatSessionEnd's agent
		// branch and closing the same cross-tenant message-injection gap
		// already fixed for the a.IsDirect() branch below.
		s, err := h.sessionGet(ctx, sessionID)
		if err != nil {
			log.Errorf("Could not validate the session info. err: %v", err)
			return nil, err
		}
		if !h.hasPermission(ctx, a, s.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			return nil, serviceerrors.ErrPermissionDenied
		}
		senderID = a.AgentID()
		// An agent/accesskey caller can only ever author an outbound
		// (business-to-visitor) reply.
		direction = wcmessage.DirectionOutbound
	case a.IsDirect():
		if !a.HasAllowedResourceType("webchat_session") {
			return nil, serviceerrors.ErrPermissionDenied
		}
		// The visitor's direct-scope JWT is bound to a single widget_id
		// (DirectScope.ResourceID). Resolve the target session and verify
		// it actually belongs to that widget before allowing message
		// injection -- otherwise any visitor JWT could post into (or
		// read the reply stream of) an arbitrary session UUID belonging
		// to a different customer's widget.
		s, err := h.sessionGet(ctx, sessionID)
		if err != nil {
			log.Errorf("Could not validate the session info. err: %v", err)
			return nil, err
		}
		if s.WidgetID != a.DirectScope.ResourceID {
			return nil, serviceerrors.ErrPermissionDenied
		}
		// A visitor holding only a direct-scope JWT can only ever author
		// an inbound (visitor-to-business) message. Without this, a
		// visitor could forge direction=outbound and have the message
		// mirrored into the agent-facing Conversation transcript as if
		// an agent had sent it (same-tenant integrity issue, not caught
		// by the ownership checks above).
		direction = wcmessage.DirectionInbound
	default:
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.WebchatV1MessageCreate(ctx, a.CustomerID, sessionID, direction, senderID, text)
	if err != nil {
		log.Errorf("Could not create the message. err: %v", err)
		return nil, err
	}
	log.WithField("message", tmp).Debug("Create a new message.")

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// WebchatMessageDelete sends a request to webchat-manager to delete the message.
func (h *serviceHandler) WebchatMessageDelete(ctx context.Context, a *auth.AuthIdentity, messageID uuid.UUID) (*wcmessage.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "WebchatMessageDelete",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	m, err := h.webchatMessageGet(ctx, messageID)
	if err != nil {
		log.Errorf("Could not get message. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, m.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.WebchatV1MessageDelete(ctx, messageID)
	if err != nil {
		log.Errorf("Could not delete the message. err: %v", err)
		return nil, err
	}
	log.WithField("message", tmp).Debugf("Deleted message. message_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
