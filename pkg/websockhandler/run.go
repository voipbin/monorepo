package websockhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/hook"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/zmqsubhandler"
)

// Run creates a new websocket and starts socket message listen.
func (h *websockHandler) Run(ctx context.Context, w http.ResponseWriter, r *http.Request, a *amagent.Agent) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "Run",
		"agent": a,
	})

	// create a websock
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("Could not create websocket. err: %v", err)
		return err
	}
	defer ws.Close()
	log.Debugf("Created a new websocket correctly.")

	// create a new subscriber zmqSub
	zmqSub, err := zmqsubhandler.NewZMQSubHandler()
	if err != nil {
		log.Errorf("Could not create a new zmq subscirber handler. err: %v", err)
		return err
	}
	defer zmqSub.Terminate()
	log.Debugf("Created a new subscribe socket correctly.")

	// we are creating a new context and cancel using the http request.
	// we are expecting when the websocket closed, everything is closed too.
	newCtx, newCancel := context.WithCancel(r.Context())
	go h.runWebsock(newCtx, newCancel, a, ws, zmqSub)
	go h.runZMQSub(newCtx, newCancel, ws, zmqSub)

	<-newCtx.Done()
	log.Debugf("Websocket connection has been closed. agent_id: %s", a.ID)

	return nil
}

// runZMQSub runs the zmq subscriber
func (h *websockHandler) runZMQSub(
	ctx context.Context,
	cancel context.CancelFunc,
	ws *websocket.Conn,
	zmqSub zmqsubhandler.ZMQSubHandler,
) {
	log := logrus.WithFields(logrus.Fields{
		"func": "runZMQSub",
	})
	defer cancel()

	if errRun := zmqSub.Run(ctx, ws); errRun != nil {
		log.Infof("The zmq subscriber run has finished. err: %v", errRun)
		return
	}
}

func (h *websockHandler) runWebsock(
	ctx context.Context,
	cancel context.CancelFunc,
	a *amagent.Agent,
	ws *websocket.Conn,
	zmqSub zmqsubhandler.ZMQSubHandler,
) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "runWebsock",
		"agent": a,
	})
	defer cancel()

	for {
		// receive the message from the subscriber
		m, err := h.receiveMessageFromWebsock(ctx, ws)
		if err != nil {
			log.Infof("Could not receive the message correctly. Assume the websocket has closed. err: %v", err)
			return
		}
		log.Debugf("Received subscribee/unsubscribe message from the websocket. type: %s, topics: %v", m.Type, m.Topics)

		// handle the message
		if errHandle := h.handleMessage(ctx, a, zmqSub, m); errHandle != nil {
			log.Errorf("Could not handle the message correctly. err: %v", errHandle)
			return
		}
	}
}

// receiveMessageFromWebsock receives the message from the websock and sends it to the channel
func (h *websockHandler) receiveMessageFromWebsock(ctx context.Context, ws *websocket.Conn) (*hook.Hook, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "receiveMessageFromWebsock",
	})

	for {
		if ctx.Err() != nil {
			log.Infof("The context is canceled. Exiting the receiving loop. err: %v", ctx.Err())
			return nil, ctx.Err()
		}

		// read the message from the websocket
		t, m, err := ws.ReadMessage()
		if err != nil {
			log.Infof("Could not read the message correctly. err: %v", err)
			return nil, err
		}
		log.Debugf("Recevied websocket message. type: %d", t)

		res := &hook.Hook{}
		if errUnmarshal := json.Unmarshal(m, res); errUnmarshal != nil {
			log.Errorf("Could not unmarshal the message correctly. err: %v", errUnmarshal)
			continue
		}

		return res, nil
	}
}

// handleMessage handles the message from the websock and do the subscription/unsubscription.
func (h *websockHandler) handleMessage(ctx context.Context, a *amagent.Agent, zmqSub zmqsubhandler.ZMQSubHandler, m *hook.Hook) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "handleMessage",
		"agent":   a,
		"message": m,
	})

	// validate the topics
	if !h.validateTopics(ctx, a, m.Topics) {
		log.Errorf("Invalid topics.")
		return fmt.Errorf("invalid topics")
	}

	switch m.Type {
	case hook.TypeSubscribe:
		for _, topic := range m.Topics {
			if errSub := zmqSub.Subscribe(topic); errSub != nil {
				log.Errorf("Could not subscribe the topic. topic: %s, err: %v", topic, errSub)
				return errSub
			}
			log.Debugf("Subscribed the topic. topic: %s", topic)
		}

	case hook.TypeUnsubscribe:
		for _, topic := range m.Topics {
			if errSub := zmqSub.Unsubscribe(topic); errSub != nil {
				log.Errorf("Could not unsubscribe the topic. topic: %s, err: %v", topic, errSub)
				return errSub
			}
			log.Debugf("Unsubscribed the topic. topic: %s", topic)
		}
	}

	return nil
}
