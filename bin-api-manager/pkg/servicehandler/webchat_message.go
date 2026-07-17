package servicehandler

import (
	"context"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"

	amagent "monorepo/bin-agent-manager/models/agent"

	wcmessage "monorepo/bin-webchat-manager/models/message"
	wcsession "monorepo/bin-webchat-manager/models/session"

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

	// Mirror flowGet/widgetGet/sessionGet's established pattern: a
	// soft-deleted message must behave as not-found -- found via round
	// 7 of an independent adversarial code review.
	if res.TMDelete != nil {
		return nil, serviceerrors.ErrNotFound
	}

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
// sessionID, if non-nil (uuid.Nil means "no filter"), scopes the list to
// messages belonging to that session -- e.g. the message timeline shown
// when an agent clicks into a session from a widget's Sessions tab in
// square-admin. The session's ownership is verified via sessionGet before
// the filter is applied, mirroring WebchatMessageCreate's
// fetch-then-check-owner pattern, so a caller cannot use an arbitrary
// session_id to enumerate another customer's messages.
//
// Also reachable by a direct-scoped caller (a widget visitor) -- added
// per VOIP-1265, mirroring WebchatMessageCreate's dual-path auth. A
// visitor can only ever list their own session's messages (session_id is
// mandatory in that path); widget soft-delete is checked (mirrors
// Create's direct branch), but an ended session's history remains
// readable (deliberately -- unlike posting into it, reading the final
// history of an ended session is a legitimate need, e.g. reconnecting
// after a network drop to see the last replies before the session
// ended).
func (h *serviceHandler) WebchatMessageList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, sessionID uuid.UUID) ([]*wcmessage.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "WebchatMessageList",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	filters := map[wcmessage.Field]any{
		wcmessage.FieldDeleted: false,
	}

	switch {
	case a.IsDirect():
		if !a.HasAllowedResourceType("webchat_session") {
			return nil, serviceerrors.ErrPermissionDenied
		}
		if sessionID == uuid.Nil {
			// A visitor must always scope to their own session; there is
			// no "list all my messages across sessions" concept for a
			// direct-scoped caller.
			return nil, serviceerrors.ErrPermissionDenied
		}
		s, err := h.sessionGet(ctx, sessionID)
		if err != nil {
			log.Errorf("Could not validate the session info. err: %v", err)
			return nil, err
		}
		if s.WidgetID != a.DirectScope.ResourceID {
			return nil, serviceerrors.ErrPermissionDenied
		}
		// Confirm the widget itself hasn't been soft-deleted -- mirrors
		// WebchatMessageCreate's/WebchatSessionCreate's direct branches.
		if _, err := h.widgetGet(ctx, a.DirectScope.ResourceID); err != nil {
			log.Errorf("Could not validate the widget info. err: %v", err)
			return nil, err
		}
		// s.CustomerID is authoritative here (already ownership-verified
		// above via s.WidgetID check) -- set explicitly rather than
		// trusting a.CustomerID, mirroring WebchatMessageCreate's
		// ownerCustomerID pattern for the same defense-in-depth reason.
		filters[wcmessage.FieldCustomerID] = s.CustomerID
		filters[wcmessage.FieldSessionID] = sessionID

	default:
		if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			log.Info("The agent has no permission.")
			return nil, serviceerrors.ErrPermissionDenied
		}

		filters[wcmessage.FieldCustomerID] = a.CustomerID

		if sessionID != uuid.Nil {
			s, err := h.sessionGet(ctx, sessionID)
			if err != nil {
				log.Errorf("Could not validate the session info. err: %v", err)
				return nil, err
			}
			if s.CustomerID != a.CustomerID {
				log.Info("The session does not belong to the requesting customer.")
				return nil, serviceerrors.ErrPermissionDenied
			}
			filters[wcmessage.FieldSessionID] = sessionID
		}
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
	// ownerCustomerID is the customer_id the created message must be
	// tagged with, resolved to the SESSION's actual owner in each switch
	// branch below rather than assumed to equal the caller's own
	// a.CustomerID -- they are normally identical, but a ProjectSuperAdmin
	// caller can pass hasPermission's ownership check while belonging to
	// a completely different customer than the session (hasPermission
	// short-circuits to true for PermissionProjectSuperAdmin regardless
	// of a.CustomerID). Passing the caller's own a.CustomerID through in
	// that case would create a message tagged with the WRONG customer_id,
	// mismatched from the session/widget's real owner -- a data-integrity
	// bug distinct from the permission-check and direction-spoofing gaps
	// fixed in earlier rounds. Mirrors the same fix applied to
	// WebchatSessionCreate.
	var ownerCustomerID uuid.UUID

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
		// Reject posting into an already-ended session -- session.go's
		// own documented lifecycle contract states "ended is terminal; a
		// subsequent message ... creates a NEW Session", but nothing in
		// this call path (or webchat-manager's messagehandler) enforced
		// that before this fix, making Status purely cosmetic. Found via
		// round 10 of an independent adversarial code review.
		if s.Status == wcsession.StatusEnded {
			return nil, serviceerrors.ErrNotFound
		}
		senderID = a.AgentID()
		// For an accesskey-authenticated caller, a.AgentID() is always
		// uuid.Nil (a.Agent is nil for Accesskey identities) -- without
		// this override, an accesskey-originated outbound reply would be
		// persisted with the same empty SenderID as a genuinely
		// automated flow/AI-originated message, losing the distinction
		// documented in message.go ("SenderID: agent user ID for an
		// agent-typed outbound reply; empty for flow/AI-originated or
		// inbound messages") and making it impossible to audit which
		// integration/accesskey sent a given reply. Mirrors the
		// ownerID-resolution pattern in storage_file.go.
		if a.IsAccesskey() {
			senderID = a.AccesskeyID()
		}
		// An agent/accesskey caller can only ever author an outbound
		// (business-to-visitor) reply.
		direction = wcmessage.DirectionOutbound
		ownerCustomerID = s.CustomerID
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
		// Same ended-session rejection as the agent/accesskey branch
		// above -- round 10 finding.
		if s.Status == wcsession.StatusEnded {
			return nil, serviceerrors.ErrNotFound
		}
		// Confirm the widget itself hasn't been soft-deleted -- a widget
		// deletion is expected to shut off the visitor-facing surface
		// entirely. Without this, a visitor holding a previously-issued
		// direct-scope JWT for a now-deleted widget could keep posting
		// new inbound messages indefinitely into any of that widget's
		// still-open sessions, making widget deletion ineffective as a
		// kill switch for ongoing message flow. Mirrors
		// WebchatSessionCreate's direct branch, which already performs
		// this same widgetGet check for the identical reason. Found via
		// round 9 of an independent adversarial code review.
		if _, err := h.widgetGet(ctx, a.DirectScope.ResourceID); err != nil {
			log.Errorf("Could not validate the widget info. err: %v", err)
			return nil, err
		}
		// Defense in depth: tag the message with the session's actual
		// owner rather than assuming a.CustomerID (derived from the
		// direct-scope JWT at boot time) still matches it.
		ownerCustomerID = s.CustomerID
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

	tmp, err := h.reqHandler.WebchatV1MessageCreate(ctx, ownerCustomerID, sessionID, direction, senderID, text)
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
