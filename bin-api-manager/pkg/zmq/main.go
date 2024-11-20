package zmq

//go:generate mockgen -package zmq -destination ./mock_zmq.go -source main.go -build_flags=-mod=mod

import (
	"github.com/pebbe/zmq4"
)

// ZMQ defines
type ZMQ interface {
	Bind(t zmq4.Type, addr string) error
	Connect(t zmq4.Type, addr string) error
	Terminate()

	Subscribe(topic string) error
	Unsubscribe(topic string) error

	Publish(topic string, m string) error
	Receive() ([]string, error)
	ReceiveNoBlock() ([]string, error)
}

// zmq struct
type zmq struct {
	sockType zmq4.Type
	address  string

	socket *zmq4.Socket
}

// NewZMQ returns a new ZMQ instance
func NewZMQ() ZMQ {
	return &zmq{}
}
