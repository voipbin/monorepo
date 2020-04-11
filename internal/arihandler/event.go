package arihandler

import (
	log "github.com/sirupsen/logrus"

	ari "gitlab.com/voipbin/bin-manager/call-manager/internal/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/internal/rabbitmq"
)

// ReceiveEventQueue create RabbitMQ Queue for ARI event receive and starts to receive event
func ReceiveEventQueue(addr, queue, receiver string) {
	// create queue for ari event receive
	log.WithFields(log.Fields{
		"addr":  addr,
		"queue": queue,
	}).Infof("Creating rabbitmq queue for ARI event receiving.")
	q := rabbitmq.NewQueue(addr, queue, true)

	// connect
	q.Connect()

	// receive ARI event
	q.ConsumeMessage(receiver, processEvent)
}

// processEvent processes received ARI event
func processEvent(m string) error {
	log.Debugf("Event recevied. event: %s", m)

	event, evt, err := ari.Parse([]byte(m))
	if err != nil {
		return err
	}
	promARIEventTotal.WithLabelValues(event.Type, event.AsteriskID).Inc()

	log.Debugf("Parsed event. evt: %v", evt)

	return nil
}
