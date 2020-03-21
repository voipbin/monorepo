package main

import (
	"flag"
	"fmt"

	// "log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

var ariAddr = flag.String("ari_addr", "localhost:8088", "asterisk ari service address")
var ariAccount = flag.String("ari_account", "asterisk:asterisk", "asterisk ari account info. id:password")
var ariSubscribeAll = flag.String("ari_subscribe_all", "true", "asterisk subscribe all.")
var ariApplication = flag.String("ari_application", "asterisk-proxy", "asterisk ari application name.")

var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")
var rabbitQueue = flag.String("rabbit_queue", "asterisk_ari", "rabbitmq queue name.")

// create message buffer
var chMessages = make(chan []byte, 1024000)

func main() {
	initiate()

	// asterisk ari message receiver
	go handleARI()

	// push the message into rabbitmq
	go handleRabbitMQ()

	forever := make(chan bool)
	<-forever
}

// initiate initiates asterisk_proxy
func initiate() {
	flag.Parse()
	log.SetFormatter(&log.JSONFormatter{})
	log.Info("asterisk-proxy started.")
}

// connectARI connects to Asterisk's ARI websocket.
// return *websocket.Conn must be closed after use.
func connectARI(account, subscribe, application string) (*websocket.Conn, error) {
	// create url query
	rawquery := fmt.Sprintf("api_key=%s&subscribeAll=%s&app=%s", account, subscribe, application)

	u := url.URL{
		Scheme:   "ws",
		Host:     *ariAddr,
		Path:     "/ari/events",
		RawQuery: rawquery,
	}
	log.Debugf("Connect to Asterisk ARI. dial string: %s", u.String())

	// connect
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// handleARIEevnt reads the event from the websocket and writes to the channel.
func recvARIEvent(c *websocket.Conn) error {
	// receive ARI events
	msgType, msgStr, err := c.ReadMessage()
	if err != nil {
		log.Errorf("Could not read message. msgType: %d, err: %v", msgType, err)
		return err
	}
	chMessages <- msgStr

	return nil
}

// handleARI handles ARI events and ARI connection
func handleARI() {
	// connect to Asterisk ARI
	for {
		conn, err := connectARI(*ariAccount, *ariSubscribeAll, *ariApplication)
		if err != nil {
			log.Errorf("Could not connect to Asterisk ARI. err: %v", err)
			time.Sleep(time.Second * 1)

			continue
		}
		defer conn.Close()

		// receive ARI events
		for {
			if err := recvARIEvent(conn); err != nil {
				log.Errorf("Could not recv the ARI event. err: %v", err)
				break
			}
		}

		time.Sleep(1 * time.Second)
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

// publishMSG watchs messages channel and send it to given rabbitMQ channel/queue.
func publishMSG(ch *amqp.Channel, q amqp.Queue) error {
	// message sending
	for {
		select {
		case msg := <-chMessages:
			// message send
			err := ch.Publish(
				"",     // excahnge
				q.Name, // routing key
				false,  // madatory
				false,
				amqp.Publishing{
					DeliveryMode: amqp.Persistent,
					ContentType:  "text/plain",
					Body:         msg,
				},
			)
			if err != nil {
				return err
			}
		}
	}
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
			*rabbitQueue, // name
			true,         // durable
			false,        // delete when unused
			false,        // exclusive
			false,        // no-wait
			nil,          // arguments
		)
		if err != nil {
			log.Errorf("Could not declare a queue. err: %v", err)

			time.Sleep(time.Second * 1)
			continue
		}

		// publish the message
		if err := publishMSG(ch, q); err != nil {
			log.Errorf("Could not publish the message. err: %v", err)

			time.Sleep(time.Second * 1)
			continue
		}
	}
}
