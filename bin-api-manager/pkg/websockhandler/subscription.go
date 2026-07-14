package websockhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"monorepo/bin-api-manager/models/auth"

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
	pingPeriod = 10 * time.Second
)

// subscriptionRun creates a new websocket and starts socket message listen.
func (h *websockHandler) subscriptionRun(ctx context.Context, w http.ResponseWriter, r *http.Request, a *auth.AuthIdentity) error {
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

	// heldPatterns tracks, per AMQP binding pattern, how many times THIS connection has
	// Acquire()d it (a double-subscribe to the same topic without an intervening unsubscribe
	// legitimately calls scopeRefCount.Acquire twice). This MUST be a counter, not a boolean
	// set: with a boolean set, a double-subscribe followed by a single unsubscribe would
	// delete the pattern from this connection's held set entirely while scopeRefCount's
	// internal refcount is still at 1, so ReleaseAll on abrupt disconnect would never release
	// it again -- a permanent per-scope bind leak until pod restart. See VOIP-1258 §9 / Task
	// 4.3 (leak found and fixed during PR #1101 round-2 review).
	heldPatterns := make(map[string]int)
	var heldMu sync.Mutex

	// we are creating a new context and cancel using the http request.
	// we are expecting when the websocket closed, everything is closed too.
	newCtx, newCancel := context.WithCancel(ctx)
	go h.subscriptionRunWebsock(newCtx, newCancel, a, ws, zmqSub, heldPatterns, &heldMu)
	go h.subscriptionRunZMQSub(newCtx, newCancel, ws, zmqSub, &writeMu)
	go h.subscriptionRunPinger(newCtx, newCancel, ws, &writeMu)

	<-newCtx.Done()
	log.Debugf("Websocket connection has been closed. agent_id: %s", a.AgentID())

	// abrupt-disconnect cleanup -- release everything this connection held, regardless of
	// whether an explicit unsubscribe message was ever received. Release once per
	// outstanding Acquire (heldPatterns is a count, not a boolean) so a double-subscribed
	// pattern is fully released, not left at refcount 1 forever.
	heldMu.Lock()
	patterns := make([]string, 0, len(heldPatterns))
	for p, count := range heldPatterns {
		for i := 0; i < count; i++ {
			patterns = append(patterns, p)
		}
	}
	heldMu.Unlock()
	h.scopeRefCount.ReleaseAll(patterns)

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
	a *auth.AuthIdentity,
	ws *websocket.Conn,
	zmqSub zmqsubhandler.ZMQSubHandler,
	heldPatterns map[string]int,
	heldMu *sync.Mutex,
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
		if errHandle := h.subscriptionHandleMessage(ctx, a, zmqSub, p, heldPatterns, heldMu); errHandle != nil {
			log.Errorf("Could not handle the message correctly. err: %v", errHandle)
			return
		}
	}
}

// subscriptionHandleMessage handles the message from the websock and do the subscription/unsubscription.
func (h *websockHandler) subscriptionHandleMessage(
	ctx context.Context,
	a *auth.AuthIdentity,
	zmqSub zmqsubhandler.ZMQSubHandler,
	m *hook.Hook,
	heldPatterns map[string]int,
	heldMu *sync.Mutex,
) error {
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

			// acquire the AMQP binding for this pattern. Non-fatal on error: the zmqSub
			// subscribe above already succeeded, so local filtering still works even if
			// broker-side AMQP scoping doesn't -- a safe degraded failure mode, not a hard
			// failure (VOIP-1258 §9).
			pattern, errConv := topicToBindPattern(topic)
			if errConv != nil {
				log.Errorf("Could not convert topic to bind pattern. topic: %s, err: %v", topic, errConv)
				continue
			}
			if errAcq := h.scopeRefCount.Acquire(pattern); errAcq != nil {
				log.Errorf("Could not acquire the AMQP binding. pattern: %s, err: %v", pattern, errAcq)
				continue
			}
			heldMu.Lock()
			heldPatterns[pattern]++
			heldMu.Unlock()
		}

	case hook.TypeUnsubscribe:
		for _, topic := range m.Topics {
			if errSub := zmqSub.Unsubscribe(topic); errSub != nil {
				log.Errorf("Could not unsubscribe the topic. topic: %s, err: %v", topic, errSub)
				return errSub
			}
			log.Debugf("Unsubscribed the topic. topic: %s", topic)

			// release the AMQP binding for this pattern
			pattern, errConv := topicToBindPattern(topic)
			if errConv != nil {
				log.Errorf("Could not convert topic to bind pattern. topic: %s, err: %v", topic, errConv)
				continue
			}
			if errRel := h.scopeRefCount.Release(pattern); errRel != nil {
				log.Errorf("Could not release the AMQP binding. pattern: %s, err: %v", pattern, errRel)
			}
			heldMu.Lock()
			if heldPatterns[pattern] > 1 {
				heldPatterns[pattern]--
			} else {
				delete(heldPatterns, pattern)
			}
			heldMu.Unlock()
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
