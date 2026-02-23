package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	tmspeaking "monorepo/bin-tts-manager/models/speaking"

	"github.com/pkg/errors"
)

// processEventTTSSpeakingStarted handles the tts-manager's speaking_started event
func (h *subscribeHandler) processEventTTSSpeakingStarted(ctx context.Context, m *sock.Event) error {
	var s tmspeaking.Speaking
	if err := json.Unmarshal([]byte(m.Data), &s); err != nil {
		return errors.Wrapf(err, "could not unmarshal the data. processEventTTSSpeakingStarted. err: %v", err)
	}

	if errEvent := h.billingHandler.EventTTSSpeakingStarted(ctx, &s); errEvent != nil {
		return errors.Wrapf(errEvent, "could not handle the event. processEventTTSSpeakingStarted. err: %v", errEvent)
	}

	return nil
}

// processEventTTSSpeakingStopped handles the tts-manager's speaking_stopped event
func (h *subscribeHandler) processEventTTSSpeakingStopped(ctx context.Context, m *sock.Event) error {
	var s tmspeaking.Speaking
	if err := json.Unmarshal([]byte(m.Data), &s); err != nil {
		return errors.Wrapf(err, "could not unmarshal the data. processEventTTSSpeakingStopped. err: %v", err)
	}

	if errEvent := h.billingHandler.EventTTSSpeakingStopped(ctx, &s); errEvent != nil {
		return errors.Wrapf(errEvent, "could not handle the event. processEventTTSSpeakingStopped. err: %v", errEvent)
	}

	return nil
}
