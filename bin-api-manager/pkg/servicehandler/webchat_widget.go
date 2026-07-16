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
	flowID uuid.UUID,
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

	tmp, err := h.reqHandler.WebchatV1WidgetCreate(ctx, a.CustomerID, name, welcomeMessage, flowID, sessionIdleTimeout, themeConfig)
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
	flowID uuid.UUID,
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

	tmp, err := h.reqHandler.WebchatV1WidgetUpdate(ctx, widgetID, name, welcomeMessage, flowID, sessionIdleTimeout, themeConfig)
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
