package subscribehandler

import (
	"context"
	"encoding/json"

	cmcall "monorepo/bin-call-manager/models/call"
	"monorepo/bin-common-handler/models/sock"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// processEventCMCallHangup handles the call-manager's call_hangup event.
func (h *subscribeHandler) processEventCMCallHangup(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMCallHangup",
		"event": m,
	})

	call := &cmcall.Call{}
	if err := json.Unmarshal([]byte(m.Data), &call); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if errEvent := h.activeflowHandler.EventCallHangup(ctx, call); errEvent != nil {
		log.Errorf("Could not handle the call hangup event. err: %v", errEvent)
		return errors.Wrap(errEvent, "Could not handle the call hangup event.")
	}

	return nil
}
