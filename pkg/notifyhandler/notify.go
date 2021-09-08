package notifyhandler

import (
	"context"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
)

func (h *notifyHandler) notifyEvent(t EventType, c WebhookMessage) {
	go h.publishEvent(t, c)
	go h.publishWebhook(t, c)
}

// NotifyCall
func (h *notifyHandler) NotifyCall(ctx context.Context, c *call.Call, t EventType) {
	log := logrus.WithFields(
		logrus.Fields{
			"call":       c,
			"evnet_type": t,
		},
	)
	log.Debugf("Sending call event. event_type: %s, call: %s", t, c.ID)

	// publish event
	h.notifyEvent(t, c)

	return
}

// NotifyRecording
func (h *notifyHandler) NotifyRecording(ctx context.Context, t EventType, c *recording.Recording) {
	log := logrus.WithFields(
		logrus.Fields{
			"recording":  c,
			"evnet_type": t,
		},
	)
	log.Debugf("Sending recording event. event_type: %s, recording: %s", t, c.ID)

	h.notifyEvent(t, c)

	return
}
