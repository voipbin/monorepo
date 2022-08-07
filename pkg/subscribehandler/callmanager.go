package subscribehandler

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processEventCMCallHungup handles the call-manager's call_hangup event.
func (h *subscribeHandler) processEventCMCallHungup(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":  "processEventCMCallHungup",
			"event": m,
		},
	)
	log.Debugf("Received call event. event: %s", m.Type)

	e := cmcall.Call{}
	if err := json.Unmarshal([]byte(m.Data), &e); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	h.queuecallHandler.Hungup(ctx, e.ID)

	return nil
}
