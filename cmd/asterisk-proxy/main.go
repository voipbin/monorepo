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
var messages = make(chan []byte, 1024000)

func main() {
	flag.Parse()
	// log.SetFlags(0)

	log.SetFormatter(&log.JSONFormatter{})

	log.Info("asterisk-proxy started.")

	// asterisk ari message receiver
	go recvEventFromAst()

	// push the message into rabbitmq
	go publishEvent()

	forever := make(chan bool)
	<-forever
}

// recvEventFromAst connects Asterisk's ARI websocket and receives ARI events.
func recvEventFromAst() {
	// create url query
	rawquery := fmt.Sprintf("api_key=%s&subscribeAll=%s&app=%s", *ariAccount, *ariSubscribeAll, *ariApplication)

	u := url.URL{
		Scheme:   "ws",
		Host:     *ariAddr,
		Path:     "/ari/events",
		RawQuery: rawquery,
	}
	log.Printf("Dial string: %s", u.String())

	for {
		// connect
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Println("Could not connect to server. err: ", err)

			// sleep for every second
			time.Sleep(1 * time.Second)
			continue
		}
		defer c.Close()

		// receiver
		for {
			msgType, msgStr, err := c.ReadMessage()
			if err != nil {
				log.Printf("Could not read message. msgType: %d, err: %s", msgType, err)
				break
			}

			// insert msg into queue
			messages <- msgStr
		}

		// sleep 1 second for reconnect
		time.Sleep(1 * time.Second)
	}
}

// push the message into rabbitmq
func publishEvent() {
	for {
		// connect to rabbit mq
		conn, err := amqp.Dial(*rabbitAddr)
		if err != nil {
			log.Printf("Could not connect to RabbitMQ. err: %s", err)

			time.Sleep(1 * time.Second)
			continue
		}
		defer conn.Close()

		// declare channel
		ch, err := conn.Channel()
		if err != nil {
			log.Printf("Could not declare a channel. err: %s", err)

			time.Sleep(1 * time.Second)
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
			log.Printf("Could not declare a queue. err: %s", err)

			time.Sleep(1 * time.Second)
			continue
		}

		// message sending
		for {
			select {
			case msg := <-messages:
				// message send
				err = ch.Publish(
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
			}
		}
	}
}
