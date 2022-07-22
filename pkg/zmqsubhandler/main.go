package zmqsubhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package zmqsubhandler -destination ./mock_zmqsubhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/zmq"
)

// ZMQSubHandler defines
type ZMQSubHandler interface {
	Terminate()

	Run(ctx context.Context, cancel context.CancelFunc, fn MessageHandle) error

	Subscribe(topic string) error
	Unsubscribe(topic string) error
}

const sockAddress = "inproc://api-manager"

// MessageHandle defines function
type MessageHandle func(topic, message string) error

// zmqSubHandler defines
type zmqSubHandler struct {
	sock zmq.ZMQ
}

// NewZMQSubHandler creates a new ZMQSubHandler
func NewZMQSubHandler() (ZMQSubHandler, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "NewZMQHandler",
	})

	sock := zmq.NewZMQ()
	res := &zmqSubHandler{
		sock: sock,
	}

	if err := res.initSock(); err != nil {
		log.Errorf("Could not initiate sock. err: %v", err)
		return nil, err
	}

	return res, nil
}
