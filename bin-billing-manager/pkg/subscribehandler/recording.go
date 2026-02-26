package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	cmrecording "monorepo/bin-call-manager/models/recording"

	"github.com/pkg/errors"
)

// processEventCMRecordingStarted handles the call-manager's recording_started event
func (h *subscribeHandler) processEventCMRecordingStarted(ctx context.Context, m *sock.Event) error {
	var r cmrecording.Recording
	if err := json.Unmarshal([]byte(m.Data), &r); err != nil {
		return errors.Wrapf(err, "could not unmarshal the data. processEventCMRecordingStarted. err: %v", err)
	}

	if errEvent := h.billingHandler.EventCMRecordingStarted(ctx, &r); errEvent != nil {
		return errors.Wrapf(errEvent, "could not handle the event. processEventCMRecordingStarted. err: %v", errEvent)
	}

	return nil
}

// processEventCMRecordingFinished handles the call-manager's recording_finished event
func (h *subscribeHandler) processEventCMRecordingFinished(ctx context.Context, m *sock.Event) error {
	var r cmrecording.Recording
	if err := json.Unmarshal([]byte(m.Data), &r); err != nil {
		return errors.Wrapf(err, "could not unmarshal the data. processEventCMRecordingFinished. err: %v", err)
	}

	if errEvent := h.billingHandler.EventCMRecordingFinished(ctx, &r); errEvent != nil {
		return errors.Wrapf(errEvent, "could not handle the event. processEventCMRecordingFinished. err: %v", errEvent)
	}

	return nil
}
