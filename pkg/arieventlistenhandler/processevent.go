package arieventlistenhandler

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
)

// processEvent processes received ARI event
func (h *ariEventListenHandler) processEvent(m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(
		logrus.Fields{
			"event": m,
		},
	)

	if m.Type != "ari_event" {
		return fmt.Errorf("wrong event type received. type: %s", m.Type)
	}

	// parse
	event, evt, err := ari.Parse([]byte(m.Data))
	if err != nil {
		return fmt.Errorf("could not parse the message. err: %v", err)

	}

	log = log.WithFields(
		logrus.Fields{
			"asterisk": event.AsteriskID,
			"type":     event.Type,
		},
	)

	log.Debugf("Received ARI event. type: %s", event.Type)
	promARIEventTotal.WithLabelValues(string(event.Type), event.AsteriskID).Inc()

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

	handler := processMap[event.Type]
	if handler == nil {
		// no handler
		return nil
	}

	start := time.Now()

	ctx := context.Background()
	err = handler(ctx, evt)
	elapsed := time.Since(start)

	promARIProcessTime.WithLabelValues(event.AsteriskID, string(event.Type)).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		return fmt.Errorf("could not process the ari event correctly. err: %v", err)
	}

	return nil
}
