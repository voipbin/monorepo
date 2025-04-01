package subscribehandler

import (
	"context"
	"encoding/json"

	cmcall "monorepo/bin-call-manager/models/call"
	"monorepo/bin-common-handler/models/sock"

	"github.com/pkg/errors"
)

// processEventCMCallProgressing handles the call-manager's call_created event
func (h *subscribeHandler) processEventCMCallProgressing(ctx context.Context, m *sock.Event) error {
	var c cmcall.Call
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		return errors.Wrapf(err, "could not unmarshal the data. processEventCMCallProgressing. err: %v", err)
	}

	if errEvent := h.billingHandler.EventCMCallProgressing(ctx, &c); errEvent != nil {
		return errors.Wrapf(errEvent, "could not handle the event. processEventCMCallProgressing. err: %v", errEvent)
	}

	return nil
}

// processEventCMCallHangup handles the call-manager's call_hangup event
func (h *subscribeHandler) processEventCMCallHangup(ctx context.Context, m *sock.Event) error {
	var c cmcall.Call
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		return errors.Wrapf(err, "could not unmarshal the data. processEventCMCallHangup. err: %v", err)
	}

	if errEvent := h.billingHandler.EventCMCallHangup(ctx, &c); errEvent != nil {
		return errors.Wrapf(errEvent, "could not handle the event. processEventCMCallHangup. err: %v", errEvent)
	}

	return nil
}
