package arihandler

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// ARIEvent is the structure for ARI event parse.
type ARIEvent struct {
	eventType   string
	application string
	asteriskID  string
	timestamp   time.Time
	event       interface{}
}

// Run runs the arihandler
func Run(addr string) {
	conn, err := amqp.Dial(addr)
	if err != nil {
		log.Errorf("Could not connect to AMQP. err: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Errorf("Could not get channel for queue. err: %v", err)
	}
	defer ch.Close()
}
