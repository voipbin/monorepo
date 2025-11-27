package subscribehandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/models/sock"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/sirupsen/logrus"
)

func (h *subscribeHandler) processEventPMPipecalcallInitialized(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventPMPipecalcallInitialized",
		"event": m,
	})
	log.Debugf("Received the pipecat-manager's pipecatcall_initialized event.")

	var evt pmpipecatcall.Pipecatcall
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	h.aicallHandler.EventPMPipecatcallInitialized(ctx, &evt)

	return nil
}
