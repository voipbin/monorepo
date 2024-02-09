package websockhandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/hook"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/zmqsubhandler"
)

// runZMQSub runs the zmq subscriber
func (h *websockHandler) runZMQSub(
	ctx context.Context,
	cancel context.CancelFunc,
	ws *websocket.Conn,
	sock zmqsubhandler.ZMQSubHandler,
) {
	log := logrus.WithFields(logrus.Fields{
		"func": "runZMQSub",
	})

	if errRun := sock.Run(ctx, cancel, func(topic, message string) error {
		if errWrite := ws.WriteMessage(websocket.TextMessage, []byte(message)); errWrite != nil {
			return errWrite
		}
		return nil
	}); errRun != nil {
		log.Errorf("Could not run the zmq sock run correctly. err: %v", errRun)
	}
}

func (h *websockHandler) runWebsock(
	ctx context.Context,
	cancel context.CancelFunc,
	a *amagent.Agent,
	ws *websocket.Conn,
	sock zmqsubhandler.ZMQSubHandler,
) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "runWebsock",
		"agent": a,
	})

	chanWS := make(chan hook.Hook)
	go func(cancel context.CancelFunc) {
		if errRecv := h.receiveMessageWebsock(ws, chanWS); errRecv != nil {
			log.Errorf("Could not receive the websock message correctly. err: %v", errRecv)
		}

		cancel()
	}(cancel)

	//nolint:staticcheck	// this is ok
main:
	for {

		select {
		case <-ctx.Done():
			break main

		case m := <-chanWS:
			switch m.Type {
			case hook.TypeSubscribe:
				for _, t := range m.Topics {
					// subscribe
					topic := fmt.Sprintf("%s:%s", a.CustomerID, t)
					if errSub := sock.Subscribe(topic); errSub != nil {
						log.Errorf("Could not subscribe the topic. topic: %s, err: %v", topic, errSub)
						continue
					}
					log.Debugf("Subscribed the topic. topic: %s", topic)
				}

			case hook.TypeUnsubscribe:
				for _, t := range m.Topics {
					// unsubscribe
					topic := fmt.Sprintf("%s:%s", a.CustomerID, t)
					if errSub := sock.Unsubscribe(topic); errSub != nil {
						log.Errorf("Could not unsubscribe the topic. topic: %s, err: %v", topic, errSub)
						continue
					}
					log.Debugf("Unsubscribed the topic. topic: %s", topic)
				}
			}
		}
	}
}

func (h *websockHandler) receiveMessageWebsock(ws *websocket.Conn, ch chan<- hook.Hook) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "receiveMessageWebsock",
	})

	for {
		// websocket m read
		t, m, err := ws.ReadMessage()
		if err != nil {
			log.Errorf("Could not read the message correctly. err: %v", err)
			return err
		}
		log.Debugf("Recevied websocket message. type: %d", t)

		msg := hook.Hook{}
		if errUnmarshal := json.Unmarshal(m, &msg); errUnmarshal != nil {
			log.Errorf("Could not unmarshal the message correctly. err: %v", errUnmarshal)
			continue
		}

		ch <- msg
	}
}
