package sockhandler

//go:generate mockgen -destination ./mock_sockhandler.go -package sockhandler -source ./main.go

import (
	"context"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"sync/atomic"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SockHandler interface {
	Connect()
	Close()

	ConsumeMessage(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, numWorkers int, messageConsume sock.CbMsgConsume) error
	ConsumeRPC(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, workerNum int, cbConsume sock.CbMsgRPC) error

	TopicCreate(name string) error
	TopicCreateWithKind(name string, kind string) error // NEW, Task 1.4

	EventPublish(topic string, key string, evt *sock.Event) error
	EventPublishWithDelay(topic string, key string, evt *sock.Event, delay int) error

	RequestPublish(ctx context.Context, queueName string, req *sock.Request) (*sock.Response, error)
	RequestPublishWithDelay(queueName string, req *sock.Request, delay int) error

	QueueCreate(name string, queueType string) error
	QueueSubscribe(name string, topic string) error
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error // NEW, Task 1.3
	QueueUnbind(name, key, exchange string, args amqp.Table) error            // NEW, Task 1.3
}

// rabbitFactory creates the RabbitMQ sock handler. Tests replace it with SetRabbitFactory.
var rabbitFactory atomic.Pointer[func(serverURI string) SockHandler]

func init() {
	SetRabbitFactory(nil)
}

func defaultRabbitFactory(serverURI string) SockHandler {
	return rabbitmqhandler.NewRabbit(serverURI)
}

// SetRabbitFactory replaces the function that NewSockHandler uses to create the RabbitMQ handler
// and returns a function that puts the previous one back. A nil f installs the default.
// It is meant for tests: call the returned function with defer or t.Cleanup, in reverse order of
// the calls, and do not run such a test in parallel with a test that expects the default.
func SetRabbitFactory(f func(serverURI string) SockHandler) (restore func()) {
	if f == nil {
		f = defaultRabbitFactory
	}

	prev := rabbitFactory.Swap(&f)
	return func() {
		rabbitFactory.Store(prev)
	}
}

func NewSockHandler(sockType sock.Type, serverURI string) SockHandler {

	switch sockType {

	case sock.TypeRabbitMQ:
		return (*rabbitFactory.Load())(serverURI)

	default:
		return nil
	}
}
