package eventhandler

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

func (h *eventHandler) eventARIRun() error {
	ctx := context.Background()

	for {
		// connect to Asterisk ARI
		err := h.eventARIConnect()
		if err != nil {
			logrus.Errorf("Could not connect to Asterisk ARI. err: %v", err)
			time.Sleep(time.Second * 1)

			continue
		}
		defer h.ariSock.Close()

		// receive ARI events
		for {
			if err := h.eventARIReceive(ctx); err != nil {
				logrus.Errorf("Could not recv the ARI event. err: %v", err)
				break
			}
		}

		// sleep 1 sec for reconnect
		time.Sleep(1 * time.Second)
	}
}

// eventARIConnect connects to Asterisk's ARI websocket.
// the handler's ariSock must be closed after use.
func (h *eventHandler) eventARIConnect() error {
	// create url query
	rawquery := fmt.Sprintf("api_key=%s&subscribeAll=%s&app=%s", h.ariAccount, h.ariSubscribeAll, h.ariApplication)

	u := url.URL{
		Scheme:   "ws",
		Host:     h.ariAddr,
		Path:     "/ari/events",
		RawQuery: rawquery,
	}
	logrus.Debugf("Connecting to Asterisk ARI. dial string: %s", u.String())

	// connect
	var err error
	h.ariSock, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	logrus.Debugf("Connected to Asterisk ARI. dial string: %s", u.String())

	return nil
}

// handleARIEevnt reads the event from the websocket and send it to the rabbitsock.
func (h *eventHandler) eventARIReceive(ctx context.Context) error {
	log := logrus.WithField("func", "eventARIReceive")

	// receive ARI events
	msgType, msgStr, err := h.ariSock.ReadMessage()
	if err != nil {
		log.Errorf("Could not read message. msgType: %d, err: %v", msgType, err)
		return err
	}
	log.Debugf("Recevied message. msgType: %s", msgStr)

	// notify
	h.notifyhandler.PublishEventRaw(ctx, "ari_event", "application/json", msgStr)
	log.WithField("event", msgStr).Debugf("Published ari event. event_type: %d", msgType)

	return nil
}
