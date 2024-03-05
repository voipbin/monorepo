package zmqpubhandler

import (
	"github.com/pebbe/zmq4"
	"github.com/sirupsen/logrus"
)

// init initiate zmq
func (h *zmqPubHandler) initSock() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "initSock",
	})

	if errBind := h.sock.Bind(zmq4.PUB, sockAddress); errBind != nil {
		log.Errorf("Could not bind the zmq socket. err: %v", errBind)
		return errBind
	}
	log.Debugf("Created a zmq socket for publish. address: %s", sockAddress)

	return nil
}

// Publish publishes the message to the subscribers
func (h *zmqPubHandler) Publish(topic string, message string) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Publish",
	})

	if err := h.sock.Publish(topic, message); err != nil {
		log.Errorf("Could not publish the message. err: %v", err)
		return err
	}

	return nil
}
