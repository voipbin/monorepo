package widgethandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-webchat-manager/models/widget"
)

// Get returns the widget.
func (h *widgetHandler) Get(ctx context.Context, id uuid.UUID) (*widget.Widget, error) {
	res, err := h.db.WidgetGet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not get widget. err: %w", err)
	}

	return res, nil
}

// List returns widgets.
func (h *widgetHandler) List(ctx context.Context, size uint64, token string, filters map[widget.Field]any) ([]*widget.Widget, error) {
	res, err := h.db.WidgetList(ctx, size, token, filters)
	if err != nil {
		return nil, fmt.Errorf("could not list widgets. err: %w", err)
	}

	return res, nil
}

// UpdateBasicInfo updates the widget's basic info.
func (h *widgetHandler) UpdateBasicInfo(
	ctx context.Context,
	id uuid.UUID,
	name string,
	welcomeMessage string,
	sessionFlowID uuid.UUID,
	messageFlowID uuid.UUID,
	sessionIdleTimeout int,
	themeConfig *widget.ThemeConfig,
) (*widget.Widget, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "UpdateBasicInfo",
		"widget_id": id,
	})
	log.Debug("Updating the widget's basic info.")

	if sessionIdleTimeout <= 0 {
		sessionIdleTimeout = widget.DefaultSessionIdleTimeout
	}

	fields := map[widget.Field]any{
		widget.FieldName:               name,
		widget.FieldWelcomeMessage:     welcomeMessage,
		widget.FieldSessionFlowID:      sessionFlowID,
		widget.FieldMessageFlowID:      messageFlowID,
		widget.FieldSessionIdleTimeout: sessionIdleTimeout,
		widget.FieldThemeConfig:        themeConfig,
	}

	if err := h.db.WidgetUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update the widget info. err: %v", err)
		return nil, err
	}

	res, err := h.db.WidgetGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated widget info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Delete deletes the widget.
func (h *widgetHandler) Delete(ctx context.Context, id uuid.UUID) (*widget.Widget, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "Delete",
		"widget_id": id,
	})
	log.Debug("Deleting the widget.")

	w, err := h.db.WidgetGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get widget info. err: %v", err)
		return nil, err
	}

	// delete direct hash via direct-manager (best-effort, don't block widget deletion)
	if w.DirectID != uuid.Nil {
		if _, errDirect := h.reqHandler.DirectV1DirectDelete(ctx, w.DirectID); errDirect != nil {
			log.Errorf("Could not delete direct hash. direct_id: %s, err: %v", w.DirectID, errDirect)
		}
	}

	if err := h.db.WidgetDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the widget info. err: %v", err)
		return nil, err
	}

	res, err := h.db.WidgetGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted widget info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// DirectHashRegenerate regenerates the widget's direct hash.
func (h *widgetHandler) DirectHashRegenerate(ctx context.Context, id uuid.UUID) (*widget.Widget, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "DirectHashRegenerate",
		"widget_id": id,
	})
	log.Debug("Regenerating the widget's direct hash.")

	w, err := h.db.WidgetGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get widget info. err: %v", err)
		return nil, err
	}

	if w.DirectID == uuid.Nil {
		return nil, fmt.Errorf("widget has no direct hash to regenerate")
	}

	d, err := h.reqHandler.DirectV1DirectRegenerate(ctx, w.DirectID)
	if err != nil {
		log.Errorf("Could not regenerate direct hash. err: %v", err)
		return nil, err
	}
	log.WithField("direct", d).Debugf("Direct hash regenerated. direct_id: %s", d.ID)

	// persist the newly regenerated hash string onto the widget --
	// direct-manager owns the value, but the widget keeps a
	// denormalized copy so API responses don't need a second
	// round-trip on every read (mirrors bin-ai-manager's
	// DirectHashRegenerate, see aihandler/direct_hash.go).
	fields := map[widget.Field]any{
		widget.FieldHash: d.Hash,
	}
	if err := h.db.WidgetUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update widget direct hash. err: %v", err)
		return nil, err
	}

	res, err := h.db.WidgetGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get widget info. err: %v", err)
		return nil, err
	}

	return res, nil
}
