package sockhandler

//go:generate mockgen -destination ./mock_sockhandler.go -package sockhandler -source ./main.go

import (
	"context"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

type SockHandler interface {
	Connect()
	Close()

	ConsumeMessage(queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, numWorkers int, messageConsume sock.CbMsgConsume) error
	ConsumeRPC(queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, workerNum int, cbConsume sock.CbMsgRPC) error

	TopicCreate(name string) error

	EventPublish(topic string, key string, evt *sock.Event) error
	EventPublishWithDelay(topic string, key string, evt *sock.Event, delay int) error

	RequestPublish(ctx context.Context, queueName string, req *sock.Request) (*sock.Response, error)
	RequestPublishWithDelay(queueName string, req *sock.Request, delay int) error

	QueueCreate(name string, queueType string) error
	QueueSubscribe(name string, topic string) error
}

func NewSockHandler(sockType sock.Type, serverURI string) SockHandler {

	switch sockType {

	case sock.TypeRabbitMQ:
		return rabbitmqhandler.NewRabbit(serverURI)

	default:
		return nil
	}
}
