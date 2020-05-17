package rabbitmq

import "github.com/streadway/amqp"

const (
	exchangeKindDelay string = "x-delayed-message"
)

// ExchangeDeclare declares an exchange
func (r *rabbit) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	if err := r.channel.ExchangeDeclare(name, kind, durable, autoDelete, internal, noWait, args); err != nil {
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
	}

	return nil
}

// ExchangeDeclareForDelay declares an exchange for delay
func (r *rabbit) ExchangeDeclareForDelay(name string, durable, autoDelete, internal, noWait bool) error {
	args := make(amqp.Table)
	args["x-delayed-type"] = "direct"

	return r.channel.ExchangeDeclare(name, exchangeKindDelay, durable, autoDelete, internal, noWait, args)
}
