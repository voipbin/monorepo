package rabbitmqhandler

import amqp "github.com/rabbitmq/amqp091-go"

const (
	exchangeKindDelay string = "x-delayed-message"
)

// ExchangeDeclare declares an exchange
func (r *rabbit) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	channel, err := r.connection.Channel()
	if err != nil {
		return err
	}

	if err := channel.ExchangeDeclare(name, kind, durable, autoDelete, internal, noWait, args); err != nil {
		return err
	}

	r.exchanges[name] = &exchange{
		name:       name,
		kind:       kind,
		durable:    durable,
		autoDelete: autoDelete,
		internal:   internal,
		noWait:     noWait,
		args:       args,

		channel: channel,
	}

	return nil
}

// ExchangeDeclareForDelay declares an exchange for delay
func (r *rabbit) ExchangeDeclareForDelay(name string, durable, autoDelete, internal, noWait bool) error {
	args := make(amqp.Table)
	args["x-delayed-type"] = "direct"

	return r.ExchangeDeclare(name, exchangeKindDelay, durable, autoDelete, internal, noWait, args)
}
