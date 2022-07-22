package zmqsubhandler

import (
	"github.com/pebbe/zmq4"
	"github.com/sirupsen/logrus"
)

// initSock initiate zmq
func (h *zmqSubHandler) initSock() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "init",
	})

	if errConnect := h.sock.Connect(zmq4.SUB, sockAddress); errConnect != nil {
		log.Errorf("Could not connect the zmq socket. err: %v", errConnect)
		return errConnect
	}
	log.Debugf("Created a zmq socket. address: %s", sockAddress)

	return nil
}

func (h *zmqSubHandler) Terminate() {
	h.sock.Terminate()
}

// Subscribe subscribes the topic.
func (h *zmqSubHandler) Subscribe(topic string) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Subscribe",
	})

	if errSub := h.sock.Subscribe(topic); errSub != nil {
		log.Errorf("Could not subscribe the topic. err: %v", errSub)
		return errSub
	}
	log.Debugf("Subsribed the topic correctly. topic: %s", topic)

	return nil
}

// Unsubscribe unsubscribes the topic.
func (h *zmqSubHandler) Unsubscribe(topic string) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Unsubscribe",
	})

	if errSub := h.sock.Unsubscribe(topic); errSub != nil {
		log.Errorf("Could not unsubscribe the topic. err: %v", errSub)
		return errSub
	}
	log.Debugf("Unsubscribe the topic correctly. topic: %s", topic)

	return nil
}

