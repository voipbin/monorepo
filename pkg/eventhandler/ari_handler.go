package eventhandler

import (
	"fmt"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/voip/asterisk-proxy/pkg/rabbitmq"
)

func (h *eventHandler) eventARIRun() error {
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
			if err := h.eventARIReceive(); err != nil {
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
func (h *eventHandler) eventARIReceive() error {
	// receive ARI events
	msgType, msgStr, err := h.ariSock.ReadMessage()
	if err != nil {
		logrus.Errorf("Could not read message. msgType: %d, err: %v", msgType, err)
		return err
	}
	logrus.Debugf("Recevied message. msgType: %s", msgStr)

	// create a event for message send
	event := &rabbitmq.Event{
		Type:     "ari_event",
		DataType: "application/json",
		Data:     msgStr,
	}

	// send it to rabbitmq
	if err := h.rabbitSock.PublishEvent(h.rabbitQueuePublishEvent, event); err != nil {
		logrus.Errorf("Could not send the message to the rabbitmq. queue: %s, err: %v", h.rabbitQueuePublishEvent, err)
		return err
	}
	logrus.Debugf("Sent message. queue: %s, msgType: %s", h.rabbitQueuePublishEvent, msgStr)

	return nil
}
