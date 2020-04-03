package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	call "gitlab.com/voipbin/bin-manager/call-manager/internal/call"

	joonix "github.com/joonix/log"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")
var rabbitQueueARI = flag.String("rabbit_queue_ari", "asterisk_ari", "rabbitmq asterisk ari queue name.")

func main() {
	fmt.Println("Hello world!")

	log.WithFields(log.Fields{
		"msg": "hello",
	}).Debug("Hello world!")

	// run workers
	go signalHandler()
	go timeClock()
	go handleRabbitMQ()

	simple := call.Call{
		ID: uuid.NewV4(),
	}
	fmt.Println(simple)

	<-chDone

	return
}

// proces init
func init() {
	flag.Parse()

	// init logs
	log.SetFormatter(joonix.NewFormatter())
	log.SetLevel(log.DebugLevel)

	// init signal handler
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGHUP)
}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	log.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

// timeClock prints current time per each seconds.
func timeClock() {
	log.Debug("timeClock started.")
	for {
		log.Debugf("Current time is: %v", time.Now())
		time.Sleep(time.Second * 1)
	}
}

// connectRabbitMQ connects to the gvien rabbitMQ address.
// returned *amqp.Connection must be closed after use.
func connectRabbitMQ(addr string) (*amqp.Connection, error) {
	// connect to rabbit mq
	conn, err := amqp.Dial(addr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// handleRabbitMQ handles rabbitMQ connection and message delivery.
func handleRabbitMQ() {
	for {
		// connect to rabbitmq
		conn, err := connectRabbitMQ(*rabbitAddr)
		if err != nil {
			log.Errorf("Could not connect to Rabbitmq. addr: %s, err: %v", *rabbitAddr, err)

			time.Sleep(1 * time.Second)
			continue
		}
		defer conn.Close()

		// create channel
		ch, err := conn.Channel()
		if err != nil {
			log.Errorf("Could not create a channel for RabbitMQ send. err: %v", err)

			time.Sleep(time.Second * 1)
			continue
		}
		defer ch.Close()

		// set queue
		q, err := ch.QueueDeclare(
			*rabbitQueueARI, // name
			true,            // durable
			false,           // delete when unused
			false,           // exclusive
			false,           // no-wait
			nil,             // arguments
		)
		if err != nil {
			log.Errorf("Could not declare a queue. err: %v", err)

			time.Sleep(time.Second * 1)
			continue
		}

		// recevie the message
		if err := receiveRabbitMSG(ch, q); err != nil {
			log.Errorf("Could not receive the message. err: %v", err)

			time.Sleep(time.Second * 1)
			continue
		}
	}
}

// receiveRabbitMSG receives rabbitmq message and prints it.
func receiveRabbitMSG(ch *amqp.Channel, q amqp.Queue) error {
	// message receiving
	for {
		msgs, err := ch.Consume(
			q.Name,
			"",
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return err
		}

		go func(deliveries <-chan amqp.Delivery) {
			for msg := range msgs {
				log.Debugf("Received message. msg: %s", msg.Body)
			}
		}(msgs)

	}
}
