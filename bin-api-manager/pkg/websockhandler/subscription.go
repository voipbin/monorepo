package websockhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/models/hook"
	"monorepo/bin-api-manager/pkg/zmqsubhandler"
)

const (
	// writeWait is the time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// pongWait is the time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// pingPeriod sends pings to peer with this period. Must be less than pongWait.
	pingPeriod = 30 * time.Second
)

// subscriptionRun creates a new websocket and starts socket message listen.
func (h *websockHandler) subscriptionRun(ctx context.Context, w http.ResponseWriter, r *http.Request, a *amagent.Agent) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "subscriptionRun",
		"agent": a,
	})

	// create a websock
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("Could not create websocket. err: %v", err)
		return err
	}
	defer func() {
		_ = ws.Close()
	}()
	log.Debugf("Created a new websocket correctly.")

	// create a mutex for protecting concurrent writes to the websocket
	var writeMu sync.Mutex

	// set up pong handler to extend read deadline on each pong received
	ws.SetPongHandler(func(string) error {
		log.Debugf("Received pong from client")
		return ws.SetReadDeadline(time.Now().Add(pongWait))
	})

	// set initial read deadline
	if err := ws.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Errorf("Could not set initial read deadline. err: %v", err)
		return err
	}

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
	newCtx, newCancel := context.WithCancel(ctx)
	go h.subscriptionRunWebsock(newCtx, newCancel, a, ws, zmqSub)
	go h.subscriptionRunZMQSub(newCtx, newCancel, ws, zmqSub, &writeMu)
	go h.subscriptionRunPinger(newCtx, newCancel, ws, &writeMu)

	<-newCtx.Done()
	log.Debugf("Websocket connection has been closed. agent_id: %s", a.ID)

	return nil
}

// subscriptionRunZMQSub runs the zmq subscriber
func (h *websockHandler) subscriptionRunZMQSub(
	ctx context.Context,
	cancel context.CancelFunc,
	ws *websocket.Conn,
	zmqSub zmqsubhandler.ZMQSubHandler,
	writeMu *sync.Mutex,
) {
	log := logrus.WithFields(logrus.Fields{
		"func": "subscriptionRunZMQSub",
	})
	defer cancel()

	if errRun := zmqSub.RunWithMutex(ctx, ws, writeMu); errRun != nil {
		log.Infof("The zmq subscriber run has finished. err: %v", errRun)
		return
	}
}

// subscriptionRunWebsock process the received message from the websocket connection
func (h *websockHandler) subscriptionRunWebsock(
	ctx context.Context,
	cancel context.CancelFunc,
	a *amagent.Agent,
	ws *websocket.Conn,
	zmqSub zmqsubhandler.ZMQSubHandler,
) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "subscriptionRunWebsock",
		"agent": a,
	})
	defer cancel()

	for {
		// receive the message from the subscriber
		m, err := h.receiveTextFromWebsock(ctx, ws)
		if err != nil {
			log.Infof("Could not receive the message correctly. Assume the websocket has closed. err: %v", err)
			return
		}

		p := &hook.Hook{}
		if errUnmarshal := json.Unmarshal(m, p); errUnmarshal != nil {
			log.Errorf("Could not unmarshal the message correctly. err: %v", errUnmarshal)
			continue
		}
		log.Debugf("Received subscribee/unsubscribe message from the websocket. type: %s, topics: %v", p.Type, p.Topics)

		// handle the message
		if errHandle := h.subscriptionHandleMessage(ctx, a, zmqSub, p); errHandle != nil {
			log.Errorf("Could not handle the message correctly. err: %v", errHandle)
			return
		}
	}
}

// subscriptionHandleMessage handles the message from the websock and do the subscription/unsubscription.
func (h *websockHandler) subscriptionHandleMessage(ctx context.Context, a *amagent.Agent, zmqSub zmqsubhandler.ZMQSubHandler, m *hook.Hook) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "subscriptionHandleMessage",
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

// subscriptionRunPinger sends periodic ping frames to keep the WebSocket connection alive
func (h *websockHandler) subscriptionRunPinger(
	ctx context.Context,
	cancel context.CancelFunc,
	ws *websocket.Conn,
	writeMu *sync.Mutex,
) {
	log := logrus.WithFields(logrus.Fields{
		"func": "subscriptionRunPinger",
	})
	defer cancel()

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			writeMu.Lock()
			if err := ws.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Infof("Could not set write deadline. err: %v", err)
				writeMu.Unlock()
				return
			}
			if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Infof("Could not write ping message. err: %v", err)
				writeMu.Unlock()
				return
			}
			writeMu.Unlock()
			log.Debugf("Sent ping message to client")

		case <-ctx.Done():
			log.Debugf("Context canceled, exiting pinger goroutine")
			return
		}
	}
}
