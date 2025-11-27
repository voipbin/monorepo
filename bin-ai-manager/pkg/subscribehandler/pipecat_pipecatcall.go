package subscribehandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/models/sock"
	pmmessage "monorepo/bin-pipecat-manager/models/message"

	"github.com/sirupsen/logrus"
)

func (h *subscribeHandler) processEventPMPipecalcallInitialized(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventPMPipecalcallInitialized",
		"event": m,
	})
	log.Debugf("Received the pipecat-manager's pipecatcall_initialized event.")

	var evt pmmessage.Message
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	h.messageHandler.EventPMMessageUserTranscription(ctx, &evt)

	return nil
}
