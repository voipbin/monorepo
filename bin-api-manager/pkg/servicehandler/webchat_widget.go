package servicehandler

import (
	"context"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"

	amagent "monorepo/bin-agent-manager/models/agent"

	wcwidget "monorepo/bin-webchat-manager/models/widget"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// widgetGet validates the widget's ownership and returns the widget info.
func (h *serviceHandler) widgetGet(ctx context.Context, id uuid.UUID) (*wcwidget.Widget, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "widgetGet",
		"widget_id": id,
	})

	res, err := h.reqHandler.WebchatV1WidgetGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the widget info. err: %v", err)
		return nil, err
	}
	log.WithField("widget", res).Debug("Received result.")

	// Mirror flowGet's established pattern: a soft-deleted widget must
	// behave as not-found to every caller resolving ownership through
	// this helper. Without this, WebchatWidgetUpdate could silently
	// resurrect a deleted widget's config, WebchatWidgetDirectHashRegenerate
	// could mint fresh visitor-facing credentials for a widget that
	// should be gone, and WebchatSessionCreate could open a brand-new
	// active session against a deleted widget -- WebchatV1WidgetGet
	// itself does not filter TMDelete (only the List path does, via
	// FieldDeleted), so this check is the only enforcement point. Found
	// via round 7 of an independent adversarial code review.
	if res.TMDelete != nil {
		return nil, serviceerrors.ErrNotFound
	}

	return res, nil
}

// WebchatWidgetGet sends a request to webchat-manager to get the widget.
func (h *serviceHandler) WebchatWidgetGet(ctx context.Context, a *auth.AuthIdentity, widgetID uuid.UUID) (*wcwidget.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "WebchatWidgetGet",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"widget_id":   widgetID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	tmp, err := h.widgetGet(ctx, widgetID)
	if err != nil {
		log.Errorf("Could not validate the widget info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// WebchatWidgetList sends a request to webchat-manager to get a list of widgets.
func (h *serviceHandler) WebchatWidgetList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*wcwidget.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "WebchatWidgetList",
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

	filters := map[wcwidget.Field]any{
		wcwidget.FieldCustomerID: a.CustomerID,
		wcwidget.FieldDeleted:    false,
	}

	tmps, err := h.reqHandler.WebchatV1WidgetList(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get widgets from the webchat-manager. err: %v", err)
		return nil, err
	}

	res := []*wcwidget.WebhookMessage{}
	for _, tmp := range tmps {
		e := tmp.ConvertWebhookMessage()
		res = append(res, e)
	}

	return res, nil
}

// WebchatWidgetCreate sends a request to webchat-manager to create a widget.
func (h *serviceHandler) WebchatWidgetCreate(
	ctx context.Context,
	a *auth.AuthIdentity,
	name string,
	welcomeMessage string,
	sessionFlowID uuid.UUID,
	messageFlowID uuid.UUID,
	sessionIdleTimeout int,
	themeConfig *wcwidget.ThemeConfig,
) (*wcwidget.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "WebchatWidgetCreate",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// Verify BOTH referenced flows (SessionFlowID and MessageFlowID,
	// independently) belong to the caller's own customer -- otherwise
	// Customer A could point their widget at Customer B's flow, and a
	// session-create/inbound-message on that widget would trigger
	// Customer B's Flow using Customer A's customer_id as the
	// activeflow owner (cross-tenant flow-execution/data-leak vector).
	// Neither bin-webchat-manager, bin-conversation-manager, nor
	// bin-flow-manager re-validates flow ownership downstream, so this
	// is the only enforcement point, mirroring CallCreate's
	// flowGet+CustomerID check in call.go. Design doc
	// 2026-07-17-webchat-widget-session-message-flow-split-design.md
	// §9: this check must run TWICE, once per field -- skipping it on
	// either field reopens the vector for that field independently.
	if sessionFlowID != uuid.Nil {
		f, err := h.flowGet(ctx, sessionFlowID)
		if err != nil {
			log.Errorf("Could not get session flow. err: %v", err)
			return nil, err
		}
		if f.CustomerID != a.CustomerID {
			log.Info("The session flow does not belong to this customer.")
			return nil, serviceerrors.ErrPermissionDenied
		}
	}
	if messageFlowID != uuid.Nil {
		f, err := h.flowGet(ctx, messageFlowID)
		if err != nil {
			log.Errorf("Could not get message flow. err: %v", err)
			return nil, err
		}
		if f.CustomerID != a.CustomerID {
			log.Info("The message flow does not belong to this customer.")
			return nil, serviceerrors.ErrPermissionDenied
		}
	}

	tmp, err := h.reqHandler.WebchatV1WidgetCreate(ctx, a.CustomerID, name, welcomeMessage, sessionFlowID, messageFlowID, sessionIdleTimeout, themeConfig)
	if err != nil {
		log.Errorf("Could not create the widget. err: %v", err)
		return nil, err
	}
	log.WithField("widget", tmp).Debug("Create a new widget.")

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// WebchatWidgetUpdate sends a request to webchat-manager to update the widget's basic info.
func (h *serviceHandler) WebchatWidgetUpdate(
	ctx context.Context,
	a *auth.AuthIdentity,
	widgetID uuid.UUID,
	name string,
	welcomeMessage string,
	sessionFlowID uuid.UUID,
	messageFlowID uuid.UUID,
	sessionIdleTimeout int,
	themeConfig *wcwidget.ThemeConfig,
) (*wcwidget.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "WebchatWidgetUpdate",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	w, err := h.widgetGet(ctx, widgetID)
	if err != nil {
		log.Errorf("Could not get widget. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, w.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// Same flow-ownership check as WebchatWidgetCreate, run TWICE (once
	// per field, design doc §9): the caller must not be able to repoint
	// an existing widget's SessionFlowID or MessageFlowID at a flow
	// belonging to a different customer.
	//
	// IMPORTANT: compare against the WIDGET's actual owner (w.CustomerID),
	// not the caller's own a.CustomerID. Unlike WebchatWidgetCreate --
	// where the new widget is always created under a.CustomerID, so the
	// two are always identical -- Update operates on an EXISTING widget
	// whose owner can legitimately differ from a.CustomerID when the
	// caller holds PermissionProjectSuperAdmin (hasPermission
	// short-circuits to true above regardless of tenant match). Using
	// a.CustomerID here let a ProjectSuperAdmin caller repoint Customer
	// B's widget (w.CustomerID) at a flow belonging to the superadmin's
	// own tenant (a.CustomerID) as long as f.CustomerID == a.CustomerID,
	// silently reintroducing the exact cross-tenant flow-execution vector
	// this check was added to close (Round 4 fix), just gated behind the
	// superadmin privilege level instead of a plain customer-admin one.
	if sessionFlowID != uuid.Nil {
		f, err := h.flowGet(ctx, sessionFlowID)
		if err != nil {
			log.Errorf("Could not get session flow. err: %v", err)
			return nil, err
		}
		if f.CustomerID != w.CustomerID {
			log.Info("The session flow does not belong to this widget's customer.")
			return nil, serviceerrors.ErrPermissionDenied
		}
	}
	if messageFlowID != uuid.Nil {
		f, err := h.flowGet(ctx, messageFlowID)
		if err != nil {
			log.Errorf("Could not get message flow. err: %v", err)
			return nil, err
		}
		if f.CustomerID != w.CustomerID {
			log.Info("The message flow does not belong to this widget's customer.")
			return nil, serviceerrors.ErrPermissionDenied
		}
	}

	tmp, err := h.reqHandler.WebchatV1WidgetUpdate(ctx, widgetID, name, welcomeMessage, sessionFlowID, messageFlowID, sessionIdleTimeout, themeConfig)
	if err != nil {
		log.Errorf("Could not update the widget. err: %v", err)
		return nil, err
	}
	log.WithField("widget", tmp).Debugf("Updated widget. widget_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// WebchatWidgetDelete sends a request to webchat-manager to delete the widget.
func (h *serviceHandler) WebchatWidgetDelete(ctx context.Context, a *auth.AuthIdentity, widgetID uuid.UUID) (*wcwidget.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "WebchatWidgetDelete",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	w, err := h.widgetGet(ctx, widgetID)
	if err != nil {
		log.Errorf("Could not get widget. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, w.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.WebchatV1WidgetDelete(ctx, widgetID)
	if err != nil {
		log.Errorf("Could not delete the widget. err: %v", err)
		return nil, err
	}
	log.WithField("widget", tmp).Debugf("Deleted widget. widget_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// WebchatWidgetDirectHashRegenerate regenerates the direct hash for the widget.
func (h *serviceHandler) WebchatWidgetDirectHashRegenerate(ctx context.Context, a *auth.AuthIdentity, widgetID uuid.UUID) (*wcwidget.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "WebchatWidgetDirectHashRegenerate",
		"customer_id": a.CustomerID,
		"widget_id":   widgetID,
	})
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	w, err := h.widgetGet(ctx, widgetID)
	if err != nil {
		log.Errorf("Could not validate the widget info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, w.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.WebchatV1WidgetDirectHashRegenerate(ctx, widgetID)
	if err != nil {
		log.Errorf("Could not regenerate widget direct hash. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
