package subscribehandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/models/sock"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *subscribeHandler) processEventPMPipecatcallInitialized(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventPMPipecatcallInitialized",
		"event": m,
	})
	log.Debugf("Received the pipecat-manager's pipecatcall_initialized event.")

	var evt pmpipecatcall.Pipecatcall
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	go h.aicallHandler.EventPMPipecatcallInitialized(ctx, &evt)

	return nil
}

// processEventPMPipecatcallTerminated dispatches pipecat-manager's
// pipecatcall_terminated event to messagehandler.EventPMPipecatcallTerminated,
// which is the cross-pod backstop that finalises any AIcall whose pipecatcall
// has terminated without an explicit terminate path completing first.
func (h *subscribeHandler) processEventPMPipecatcallTerminated(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventPMPipecatcallTerminated",
		"event": m,
	})
	log.Debugf("Received the pipecat-manager's pipecatcall_terminated event.")

	var evt pmpipecatcall.Pipecatcall
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return errors.Wrap(err, "could not unmarshal pipecatcall_terminated payload")
	}

	return h.messageHandler.EventPMPipecatcallTerminated(ctx, &evt)
}
