package servicehandler

import (
	"context"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"

	amagent "monorepo/bin-agent-manager/models/agent"

	wcsession "monorepo/bin-webchat-manager/models/session"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// sessionGet validates the session's ownership and returns the session info.
func (h *serviceHandler) sessionGet(ctx context.Context, id uuid.UUID) (*wcsession.Session, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "sessionGet",
		"session_id": id,
	})

	res, err := h.reqHandler.WebchatV1SessionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the session info. err: %v", err)
		return nil, err
	}
	log.WithField("session", res).Debug("Received result.")

	return res, nil
}

// WebchatSessionGet sends a request to webchat-manager to get the session.
func (h *serviceHandler) WebchatSessionGet(ctx context.Context, a *auth.AuthIdentity, sessionID uuid.UUID) (*wcsession.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "WebchatSessionGet",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"session_id":  sessionID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	tmp, err := h.sessionGet(ctx, sessionID)
	if err != nil {
		log.Errorf("Could not validate the session info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// WebchatSessionList sends a request to webchat-manager to get a list of sessions.
func (h *serviceHandler) WebchatSessionList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*wcsession.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "WebchatSessionList",
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

	filters := map[wcsession.Field]any{
		wcsession.FieldCustomerID: a.CustomerID,
		wcsession.FieldDeleted:    false,
	}

	tmps, err := h.reqHandler.WebchatV1SessionList(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get sessions from the webchat-manager. err: %v", err)
		return nil, err
	}

	res := []*wcsession.WebhookMessage{}
	for _, tmp := range tmps {
		e := tmp.ConvertWebhookMessage()
		res = append(res, e)
	}

	return res, nil
}

// WebchatSessionCreate sends a request to webchat-manager to create a session.
// This is the anonymous-visitor session-creation path (design doc §7/§14
// v7): called with a direct-scope JWT issued to a Widget via
// POST /auth/boot, never by a customer-JWT-authenticated agent. Also
// reachable by an authenticated agent/accesskey for admin-side testing,
// mirroring aicall.go's dual-path pattern.
func (h *serviceHandler) WebchatSessionCreate(ctx context.Context, a *auth.AuthIdentity, widgetID uuid.UUID) (*wcsession.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "WebchatSessionCreate",
		"customer_id": a.CustomerID,
		"widget_id":   widgetID,
	})

	// ownerCustomerID is the customer_id the created session must be
	// tagged with, resolved to the WIDGET's actual owner in each switch
	// branch below rather than assumed to equal the caller's own
	// a.CustomerID -- they are normally identical, but a ProjectSuperAdmin
	// caller can pass hasPermission's ownership check below while
	// belonging to a completely different customer than the widget
	// (hasPermission short-circuits to true for
	// PermissionProjectSuperAdmin regardless of a.CustomerID). Passing
	// the caller's own a.CustomerID through in that case would create a
	// session tagged with the WRONG customer_id (mismatched from
	// widget_id's real owner), making it invisible to that owner's
	// WebchatSessionList/Get calls and effectively leaking the record
	// under the superadmin's own tenant -- a data-integrity bug distinct
	// from the permission-check gaps fixed in earlier rounds. Mirrors
	// AIcallCreate's use of the resolved resource's customerID (not
	// a.CustomerID) in aicall.go.
	var ownerCustomerID uuid.UUID

	switch {
	case a.IsAgent() || a.IsAccesskey():
		// Resolve the target widget first and check permission against
		// the WIDGET's actual owner (w.CustomerID), not the caller's own
		// a.CustomerID -- the latter is a tautology that lets a Customer
		// A admin create a session (and thereby trigger Customer B's
		// configured Flow on first inbound message) for a widgetID
		// belonging to Customer B, mirroring WebchatWidgetUpdate/Delete's
		// fetch-then-check-owner pattern.
		w, err := h.widgetGet(ctx, widgetID)
		if err != nil {
			log.Errorf("Could not validate the widget info. err: %v", err)
			return nil, err
		}
		if !h.hasPermission(ctx, a, w.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			return nil, serviceerrors.ErrPermissionDenied
		}
		ownerCustomerID = w.CustomerID
	case a.IsDirect():
		if !a.HasAllowedResourceType("webchat_session") {
			return nil, serviceerrors.ErrPermissionDenied
		}
		if a.DirectScope.ResourceID != widgetID {
			return nil, serviceerrors.ErrPermissionDenied
		}
		// Defense in depth: re-resolve ownerCustomerID from the widget's
		// actual current owner rather than trusting the CustomerID claim
		// baked into the direct-scope JWT at boot time -- mirrors
		// WebchatMessageCreate's direct branch, which does the same for
		// ownerCustomerID = s.CustomerID. Also confirms the widget still
		// exists (widgetGet returns ErrNotFound for a deleted widget)
		// before creating a session against it.
		w, err := h.widgetGet(ctx, widgetID)
		if err != nil {
			log.Errorf("Could not validate the widget info. err: %v", err)
			return nil, err
		}
		ownerCustomerID = w.CustomerID
	default:
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.WebchatV1SessionCreate(ctx, ownerCustomerID, widgetID)
	if err != nil {
		log.Errorf("Could not create the session. err: %v", err)
		return nil, err
	}
	log.WithField("session", tmp).Debug("Create a new session.")

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// WebchatSessionDelete sends a request to webchat-manager to delete the session.
func (h *serviceHandler) WebchatSessionDelete(ctx context.Context, a *auth.AuthIdentity, sessionID uuid.UUID) (*wcsession.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "WebchatSessionDelete",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	s, err := h.sessionGet(ctx, sessionID)
	if err != nil {
		log.Errorf("Could not get session. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, s.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.WebchatV1SessionDelete(ctx, sessionID)
	if err != nil {
		log.Errorf("Could not delete the session. err: %v", err)
		return nil, err
	}
	log.WithField("session", tmp).Debugf("Deleted session. session_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// WebchatSessionEnd sends a request to webchat-manager to explicitly end the session.
// Reachable by both the widget's direct-scope JWT (visitor-initiated end)
// and an authenticated agent/accesskey (admin-initiated end), mirroring
// WebchatSessionCreate's dual-path auth.
func (h *serviceHandler) WebchatSessionEnd(ctx context.Context, a *auth.AuthIdentity, sessionID uuid.UUID) (*wcsession.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "WebchatSessionEnd",
		"customer_id": a.CustomerID,
		"session_id":  sessionID,
	})

	switch {
	case a.IsAgent() || a.IsAccesskey():
		s, err := h.sessionGet(ctx, sessionID)
		if err != nil {
			log.Errorf("Could not validate the session info. err: %v", err)
			return nil, err
		}
		if !h.hasPermission(ctx, a, s.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			return nil, serviceerrors.ErrPermissionDenied
		}
	case a.IsDirect():
		if !a.HasAllowedResourceType("webchat_session") {
			return nil, serviceerrors.ErrPermissionDenied
		}
		// Same widget-ownership check as WebchatMessageCreate: a visitor
		// JWT is scoped to one widget_id and must not be able to end an
		// arbitrary session belonging to a different customer's widget.
		s, err := h.sessionGet(ctx, sessionID)
		if err != nil {
			log.Errorf("Could not validate the session info. err: %v", err)
			return nil, err
		}
		if s.WidgetID != a.DirectScope.ResourceID {
			return nil, serviceerrors.ErrPermissionDenied
		}
	default:
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.WebchatV1SessionEnd(ctx, sessionID)
	if err != nil {
		log.Errorf("Could not end the session. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
