package subscribehandler

import (
	"context"
	"encoding/json"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	"monorepo/bin-common-handler/models/sock"

	"github.com/pkg/errors"
)

// processEventCMConfbridgeJoined handles the call-manager's call related event
func (h *subscribeHandler) processEventCMConfbridgeJoined(ctx context.Context, m *sock.Event) error {
	evt := cmconfbridge.EventConfbridgeJoined{}
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		return errors.Wrapf(err, "Could not unmarshal the data")
	}

	go h.aicallHandler.EventCMConfbridgeJoined(context.Background(), &evt)

	return nil
}

// processEventCMConfbridgeLeaved handles the call-manager's call related event
func (h *subscribeHandler) processEventCMConfbridgeLeaved(ctx context.Context, m *sock.Event) error {
	evt := cmconfbridge.EventConfbridgeLeaved{}
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		return errors.Wrapf(err, "Could not unmarshal the data")
	}

	go h.aicallHandler.EventCMConfbridgeLeaved(context.Background(), &evt)

	return nil
}

// processEventCMCallHangup handles the call-manager's call hangup event
func (h *subscribeHandler) processEventCMCallHangup(ctx context.Context, m *sock.Event) error {
	evt := cmcall.Call{}
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		return errors.Wrapf(err, "Could not unmarshal the data")
	}

	go h.aicallHandler.EventCMCallHangup(context.Background(), &evt)
	go h.summaryHandler.EventCMCallHangup(context.Background(), &evt)

	return nil
}
