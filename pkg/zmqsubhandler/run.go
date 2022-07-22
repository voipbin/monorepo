package zmqsubhandler

import (
	"context"

	"github.com/sirupsen/logrus"
)

type subMessage struct {
	topic   string
	message string
}

// Run runs the pubsub subscriber. It waits for the message from the sock and runs the given message handler
func (h *zmqSubHandler) Run(ctx context.Context, cancel context.CancelFunc, fn MessageHandle) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Run",
	})

	ch := make(chan subMessage)
	go func(cancel context.CancelFunc) {
		if errRecv := h.recevieMessage(ch); errRecv != nil {
			log.Errorf("Could not receive the zmq message. err: %v", errRecv)
		}

		cancel()
	}(cancel)

main:
	for {
		select {
		case <-ctx.Done():
			break main

		case m := <-ch:
			if errFn := fn(m.topic, m.message); errFn != nil {
				log.Errorf("Could not handle the message correctly. err: %v", errFn)
			}
		}
	}

	return nil
}

// recevieMessage receives the message from the socket.
func (h *zmqSubHandler) recevieMessage(ch chan<- subMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "recevieMessage",
	})

	for {
		m, err := h.sock.Receive()
		if err != nil {
			log.Errorf("Could not receive the message. err: %v", err)
			return err
		}

		if len(m) != 2 {
			log.Errorf("Received wrong type of message. message: %v", m)
			continue
		}

		tmp := subMessage{
			topic:   m[0],
			message: m[1],
		}

		ch <- tmp
	}
}
