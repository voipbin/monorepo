package servicehandler

import (
	"context"
	"fmt"
	"strings"

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

	// Mirror flowGet/widgetGet's established pattern: a soft-deleted
	// session must behave as not-found. Without this, a caller could
	// keep posting messages into or double-ending an already-deleted
	// session -- found via round 7 of an independent adversarial code
	// review.
	if res.TMDelete != nil {
		return nil, serviceerrors.ErrNotFound
	}

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
// widgetID, if non-nil (uuid.Nil means "no filter"), scopes the list to
// sessions belonging to that widget -- e.g. the Sessions tab on a widget's
// detail page in square-admin. The widget's ownership is verified via
// widgetGet before the filter is applied, mirroring
// WebchatSessionCreate/End's fetch-then-check-owner pattern, so a caller
// cannot use an arbitrary widget_id to enumerate another customer's
// sessions.
func (h *serviceHandler) WebchatSessionList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, widgetID uuid.UUID) ([]*wcsession.WebhookMessage, error) {
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

	if widgetID != uuid.Nil {
		w, err := h.widgetGet(ctx, widgetID)
		if err != nil {
			log.Errorf("Could not validate the widget info. err: %v", err)
			return nil, err
		}
		if w.CustomerID != a.CustomerID {
			log.Info("The widget does not belong to the requesting customer.")
			return nil, serviceerrors.ErrPermissionDenied
		}
		filters[wcsession.FieldWidgetID] = widgetID
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
func (h *serviceHandler) WebchatSessionCreate(ctx context.Context, a *auth.AuthIdentity, widgetID uuid.UUID, pageURL string, referrer string) (*wcsession.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "WebchatSessionCreate",
		"customer_id": a.CustomerID,
		"widget_id":   widgetID,
	})

	if err := validatePageURL(pageURL); err != nil {
		log.Infof("Invalid page_url. err: %v", err)
		return nil, err
	}

	if err := validateReferrer(referrer); err != nil {
		log.Infof("Invalid referrer. err: %v", err)
		return nil, err
	}

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

	tmp, err := h.reqHandler.WebchatV1SessionCreate(ctx, ownerCustomerID, widgetID, pageURL, referrer)
	if err != nil {
		log.Errorf("Could not create the session. err: %v", err)
		return nil, err
	}
	log.WithField("session", tmp).Debug("Create a new session.")

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// validatePageURL enforces the page_url length cap (mirrors
// validateDelegateReason's reject-don't-truncate precedent in
// auth_delegate.go) and a scheme allowlist (mirrors
// widget.go's ThemeConfig.LogoURL https-only precedent, relaxed to also
// allow http since a customer's own site is not guaranteed to be TLS).
// An empty pageURL is always valid -- the field is optional (see design
// doc §4.2).
//
// PR review round-1 finding (fixed): the design doc's edge-case analysis
// (§5) argued no scheme allowlist was needed because
// window.location.href can never itself be a javascript:/data: URL.
// That argument only holds for the JS embed runtime's own call site --
// it does NOT hold at this API boundary. WebchatSessionCreate's
// a.IsDirect() branch is reachable by ANY HTTP caller holding a widget's
// public direct-hash (see auth.md's "Service Agent Auth" / direct-scope
// JWT flow), not only the bundled client.js -- a crafted request can set
// page_url to "javascript:alert(1)" directly. That value would then be
// persisted verbatim and rendered as a clickable href in
// message_timeline.js, which does not sanitize the scheme either --
// together an admin-facing self-XSS vector (VoIPBin admin clicking a
// malicious "Started from" link inside their own square-admin session).
// Rejecting non-http(s) schemes here closes it before the value is ever
// stored.
func validatePageURL(pageURL string) error {
	if len(pageURL) > 2048 {
		return fmt.Errorf("%w: page_url exceeds maximum length of 2048 characters", serviceerrors.ErrInvalidArgument)
	}
	if pageURL == "" {
		return nil
	}
	if !strings.HasPrefix(pageURL, "http://") && !strings.HasPrefix(pageURL, "https://") {
		return fmt.Errorf("%w: page_url must use the http or https scheme", serviceerrors.ErrInvalidArgument)
	}
	return nil
}

// validateReferrer enforces the referrer length cap and scheme allowlist,
// mirroring validatePageURL exactly -- referrer is document.referrer
// captured client-side at session-creation time and is reachable through
// the same a.IsDirect() HTTP boundary as page_url (see validatePageURL's
// doc comment), so it carries the identical javascript:/data: self-XSS
// risk and must be rejected the same way. An empty referrer is always
// valid -- the field is optional.
func validateReferrer(referrer string) error {
	if len(referrer) > 2048 {
		return fmt.Errorf("%w: referrer exceeds maximum length of 2048 characters", serviceerrors.ErrInvalidArgument)
	}
	if referrer == "" {
		return nil
	}
	if !strings.HasPrefix(referrer, "http://") && !strings.HasPrefix(referrer, "https://") {
		return fmt.Errorf("%w: referrer must use the http or https scheme", serviceerrors.ErrInvalidArgument)
	}
	return nil
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
		// Confirm the widget itself hasn't been soft-deleted, mirroring
		// WebchatMessageCreate's/WebchatSessionCreate's direct branches
		// -- kept for consistency even though ending a session is a
		// terminal, low-risk operation. Found via round 9 of an
		// independent adversarial code review.
		if _, err := h.widgetGet(ctx, a.DirectScope.ResourceID); err != nil {
			log.Errorf("Could not validate the widget info. err: %v", err)
			return nil, err
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
