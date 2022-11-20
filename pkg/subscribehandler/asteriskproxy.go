package subscribehandler

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
)

// processEventAsteriskProxy handles the events from the asterisk-proxy.
func (h *subscribeHandler) processEventAsteriskProxy(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "processEventAsteriskProxy",
			"publisher": m.Publisher,
			"type":      m.Type,
		},
	)

	// parse
	event, evt, err := ari.Parse([]byte(m.Data))
	if err != nil {
		log.Errorf("Could not parse the ari event. err: %v", err)
		return errors.Wrapf(err, "could not parse the message")
	}
	log = log.WithFields(logrus.Fields{"event_type": event.Type})
	log.WithField("event", event).Debugf("Received ARI event. event_type: %s", event.Type)

	// processMap maps ARIEvent name and event handler.
	var processMap = map[ari.EventType]func(context.Context, interface{}) error{
		ari.EventTypeBridgeCreated:        h.ariEventHandler.EventHandlerBridgeCreated,
		ari.EventTypeBridgeDestroyed:      h.ariEventHandler.EventHandlerBridgeDestroyed,
		ari.EventTypeChannelCreated:       h.ariEventHandler.EventHandlerChannelCreated,
		ari.EventTypeChannelDestroyed:     h.ariEventHandler.EventHandlerChannelDestroyed,
		ari.EventTypeChannelDtmfReceived:  h.ariEventHandler.EventHandlerChannelDtmfReceived,
		ari.EventTypeChannelEnteredBridge: h.ariEventHandler.EventHandlerChannelEnteredBridge,
		ari.EventTypeChannelLeftBridge:    h.ariEventHandler.EventHandlerChannelLeftBridge,
		ari.EventTypeChannelStateChange:   h.ariEventHandler.EventHandlerChannelStateChange,
		ari.EventTypeChannelVarset:        h.ariEventHandler.EventHandlerChannelVarset,
		ari.EventTypeContactStatusChange:  h.ariEventHandler.EventHandlerContactStatusChange,
		ari.EventTypePlaybackStarted:      h.ariEventHandler.EventHandlerPlaybackStarted,
		ari.EventTypePlaybackFinished:     h.ariEventHandler.EventHandlerPlaybackFinished,
		ari.EventTypeRecordingFinished:    h.ariEventHandler.EventHandlerRecordingFinished,
		ari.EventTypeRecordingStarted:     h.ariEventHandler.EventHandlerRecordingStarted,
		ari.EventTypeStasisEnd:            h.ariEventHandler.EventHandlerStasisEnd,
		ari.EventTypeStasisStart:          h.ariEventHandler.EventHandlerStasisStart,
	}

	// get handler
	handler := processMap[event.Type]
	if handler == nil {
		// no handler
		log.Debugf("No handler registered for event. event_type: %s", event.Type)
		return nil
	}

	// execute handler
	start := time.Now()
	if errHandler := handler(ctx, evt); errHandler != nil {
		log.Errorf("Could not handle the asterisk-proxy event. err: %v", err)
	}
	elapsed := time.Since(start)
	promARIProcessTime.WithLabelValues(event.AsteriskID, string(event.Type)).Observe(float64(elapsed.Milliseconds()))

	return nil
}
