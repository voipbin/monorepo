package zmqsubhandler

import (
	"context"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Run runs the pubsub subscriber. It waits for the message from the sock and runs the given message handler
func (h *zmqSubHandler) Run(ctx context.Context, ws *websocket.Conn) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Run",
	})

	for {
		// receive the message from the subscriber
		topic, message, err := h.receiveMessage(ctx)
		if err != nil {
			log.Infof("Could not receive the message correctly. err: %v", err)
			return errors.Wrapf(err, "could not receive the message correctly")
		}
		log.Debugf("Received message from the pubsub subscriber. topic: %s", topic)

		// send the message to the websocket
		if errWrite := ws.WriteMessage(websocket.TextMessage, []byte(message)); errWrite != nil {
			log.Infof("Could not write the message to the websocket correctly. err: %v", errWrite)
			return errors.Wrapf(errWrite, "could not write the message to the websocket correctly")
		}
	}
}

// RunWithMutex runs the pubsub subscriber with mutex protection for websocket writes
func (h *zmqSubHandler) RunWithMutex(ctx context.Context, ws *websocket.Conn, writeMu *sync.Mutex) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "RunWithMutex",
	})

	for {
		// receive the message from the subscriber
		topic, message, err := h.receiveMessage(ctx)
		if err != nil {
			log.Infof("Could not receive the message correctly. err: %v", err)
			return errors.Wrapf(err, "could not receive the message correctly")
		}
		log.Debugf("Received message from the pubsub subscriber. topic: %s", topic)

		// send the message to the websocket with mutex protection
		writeMu.Lock()
		errWrite := ws.WriteMessage(websocket.TextMessage, []byte(message))
		writeMu.Unlock()

		if errWrite != nil {
			log.Infof("Could not write the message to the websocket correctly. err: %v", errWrite)
			return errors.Wrapf(errWrite, "could not write the message to the websocket correctly")
		}
	}
}

// receiveMessage receives the message from the zmq subscriber socket.
// it waits(blocks) until the message is received and returns.
func (h *zmqSubHandler) receiveMessage(ctx context.Context) (string, string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "receiveMessage",
	})

	for {
		if ctx.Err() != nil {
			log.Infof("The context is canceled. Exiting the receiving loop. err: %v", ctx.Err())
			return "", "", ctx.Err()
		}

		// note:
		// do not use the sock.Receive() here.
		// it's blocking function and it doesn't not recognize if the socket is closed or not while it's receiving.
		m, err := h.sock.ReceiveNoBlock()
		if err != nil {
			if err.Error() == syscall.EAGAIN.Error() {
				// no received message
				// wait for 1 second and try again
				time.Sleep(time.Millisecond * 1000)
				continue
			}

			log.Infof("Could not receive the message. err: %v", err)
			return "", "", errors.Wrapf(err, "could not receive the message")
		}

		if len(m) != 2 {
			// received something, but not the right type of message
			log.Errorf("Received wrong type of message. message: %v", m)
			continue
		}

		// returns topic, message, error
		return m[0], m[1], nil
	}
}
