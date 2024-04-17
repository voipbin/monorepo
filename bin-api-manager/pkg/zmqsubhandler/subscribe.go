package zmqsubhandler

import (
	"github.com/pebbe/zmq4"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
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
	log.Debugf("Created a zmq socket for subscription. address: %s", sockAddress)

	return nil
}

func (h *zmqSubHandler) Terminate() {
	log := logrus.WithFields(logrus.Fields{
		"func": "Terminate",
	})

	h.sock.Terminate()
	log.Debug("Terminated the zmq socket.")
}

// Subscribe subscribes the topic.
func (h *zmqSubHandler) Subscribe(topic string) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Subscribe",
	})

	// check already subscribed topic
	if slices.Contains(h.topics, topic) {
		log.Debugf("The topic already subscribed. topic: %s", topic)
		return nil
	}

	if errSub := h.sock.Subscribe(topic); errSub != nil {
		log.Errorf("Could not subscribe the topic. err: %v", errSub)
		return errSub
	}
	log.Debugf("Subsribed the topic correctly. topic: %s", topic)

	// add the topic
	h.topics = append(h.topics, topic)

	return nil
}

// Unsubscribe unsubscribes the topic.
func (h *zmqSubHandler) Unsubscribe(topic string) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Unsubscribe",
	})

	// delete topic from the subscribed list
	idx := -1
	for i, sub := range h.topics {
		if sub == topic {
			idx = i
			break
		}
	}
	if idx == -1 {
		// nothing to unsubscribe
		return nil
	}
	h.topics = slices.Delete(h.topics, idx, 1)

	if errSub := h.sock.Unsubscribe(topic); errSub != nil {
		log.Errorf("Could not unsubscribe the topic. err: %v", errSub)
		return errSub
	}
	log.Debugf("Unsubscribe the topic correctly. topic: %s", topic)

	return nil
}
