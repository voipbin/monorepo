package pubsubhandler

import (
	"context"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Run runs the subscriber delivery loop. It waits for matched messages and
// writes them to the given websocket.
func (s *subHandler) Run(ctx context.Context, ws *websocket.Conn) error {
	return s.run(ctx, ws, nil)
}

// RunWithMutex runs the subscriber delivery loop with mutex protection for
// websocket writes shared with other writers (e.g. the pinger goroutine).
func (s *subHandler) RunWithMutex(ctx context.Context, ws *websocket.Conn, writeMu *sync.Mutex) error {
	return s.run(ctx, ws, writeMu)
}

// run is the shared delivery loop. It returns when the context is canceled,
// the subscriber is terminated, or a websocket write fails.
//
// Note: after Terminate closes the done channel, the select may still drain a
// few already-buffered messages to the closing websocket before observing
// done. This is harmless: write errors are tolerated (they end the loop), and
// it matches the behavior of the previous ZeroMQ receive loop.
func (s *subHandler) run(ctx context.Context, ws *websocket.Conn, writeMu *sync.Mutex) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "run",
	})

	for {
		select {
		case <-ctx.Done():
			log.Infof("The context is canceled. Exiting the delivery loop. err: %v", ctx.Err())
			return ctx.Err()

		case <-s.done:
			log.Debugf("The subscriber has been terminated. Exiting the delivery loop.")
			return nil

		case m := <-s.ch:
			log.Debugf("Received message from the pubsub broker. topic: %s", m.topic)

			if writeMu != nil {
				writeMu.Lock()
			}
			errWrite := ws.WriteMessage(websocket.TextMessage, []byte(m.data))
			if writeMu != nil {
				writeMu.Unlock()
			}

			if errWrite != nil {
				log.Infof("Could not write the message to the websocket correctly. err: %v", errWrite)
				return errors.Wrapf(errWrite, "could not write the message to the websocket correctly")
			}
		}
	}
}
