package subscribehandler

import (
	"context"
	"encoding/json"

	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processEventCMCallHangup handles the call-manager's call_hangup event.
func (h *subscribeHandler) processEventCMCallHangup(ctx context.Context, m *rabbitmqhandler.Event) error {
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
