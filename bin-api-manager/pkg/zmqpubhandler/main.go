package zmqpubhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package zmqpubhandler -destination ./mock_zmqpubhandler.go -source main.go -build_flags=-mod=mod

import (
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/pkg/zmq"
)

// ZMQPubHandler defines
type ZMQPubHandler interface {
	Publish(topic string, m string) error
}

const sockAddress = "inproc://api-manager"

// MessageHandle defines function
type MessageHandle func(topic, message string) error

type zmqPubHandler struct {
	sock zmq.ZMQ
}

// NewZMQPubHandler creates a new ZMQPubHandler
func NewZMQPubHandler() ZMQPubHandler {
	log := logrus.WithFields(logrus.Fields{
		"func": "NewZMQPubHandler",
	})

	sock := zmq.NewZMQ()
	res := &zmqPubHandler{
		sock: sock,
	}

	if err := res.initSock(); err != nil {
		log.Errorf("Could not initiate sock. err: %v", err)
		return nil
	}

	return res
}
