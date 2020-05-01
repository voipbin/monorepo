package main

import (
	"flag"
	"fmt"
	"log/syslog"
	"net/url"
	"time"

	"gitlab.com/voipbin/voip/asterisk-proxy/internal/rabbitmq"
	"gitlab.com/voipbin/voip/asterisk-proxy/internal/rpc"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	lSyslog "github.com/sirupsen/logrus/hooks/syslog"

	"github.com/streadway/amqp"
)

var ariAddr = flag.String("ari_addr", "localhost:8088", "The asterisk-proxy connects to this asterisk ari service address")
var ariAccount = flag.String("ari_account", "asterisk:asterisk", "The asterisk-proxy uses this asterisk ari account info. id:password")
var ariSubscribeAll = flag.String("ari_subscribe_all", "true", "The asterisk-proxy uses this asterisk subscribe all option.")
var ariApplication = flag.String("ari_application", "asterisk-proxy", "The asterisk-proxy uses this asterisk ari application name.")

var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "The asterisk-proxy connect to rabbitmq address.")
var rabbitQueueARIEvent = flag.String("rabbit_queue_arievent", "asterisk_ari_event", "The asterisk-proxy sends the ARI event to this rabbitmq queue name.")
var rabbitQueueARIRequest = flag.String("rabbit_queue_arirequest", "asterisk_ari_request-<asterisk id>", "The asterisk-proxy gets the ARI request from this rabbitmq queue name.")

// create message buffer
var chARIEvent = make(chan []byte, 1024000)
var chRabbitMQReconnect = make(chan bool)

// logger for global
// var log = logrus.New()

func main() {
	initProcess()

	// asterisk ari message receiver
	go recevieARIEvent()

	// push the message into rabbitmq
	go handleARIEvent()

	// handle ARI request to Asterisk
	go handleARIRequest()

	forever := make(chan bool)
	<-forever
}

func initProcess() {
	// initiate flags
	flag.Parse()

	// initiate log
	log.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true, DisableColors: true})
	log.SetLevel(logrus.DebugLevel)
	hook, err := lSyslog.NewSyslogHook("", "", syslog.LOG_INFO, "")
	if err == nil {
		log.AddHook(hook)
		// log.Hooks.Add(hook)
	}

	rabbitmq.Initiate()
	rpc.Initiate(*ariAddr, *ariAccount)

	log.Info("asterisk-proxy has initiated.")
}

// connectARI connects to Asterisk's ARI websocket.
// return *websocket.Conn must be closed after use.
func connectARI(addr, account, subscribe, application string) (*websocket.Conn, error) {
	// create url query
	rawquery := fmt.Sprintf("api_key=%s&subscribeAll=%s&app=%s", account, subscribe, application)

	u := url.URL{
		Scheme:   "ws",
		Host:     addr,
		Path:     "/ari/events",
		RawQuery: rawquery,
	}
	log.Debugf("Connecting to Asterisk ARI. dial string: %s", u.String())

	// connect
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}
	log.Debugf("Connected to Asterisk ARI. dial string: %s", u.String())

	return conn, nil
}

// handleARIEevnt reads the event from the websocket and writes to the channel.
// Asterisk ARI -> Internal channel
func recvARIEvent(c *websocket.Conn) error {
	// receive ARI events
	msgType, msgStr, err := c.ReadMessage()
	if err != nil {
		log.Errorf("Could not read message. msgType: %d, err: %v", msgType, err)
		return err
	}
	chARIEvent <- msgStr

	return nil
}

// recevieARIEvent handles ARI events and ARI connection
func recevieARIEvent() {
	// connect to Asterisk ARI
	for {
		conn, err := connectARI(*ariAddr, *ariAccount, *ariSubscribeAll, *ariApplication)
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
func connectRabbitMQ(addr string) *amqp.Connection {
	for {
		// connect to rabbit mq
		conn, err := amqp.Dial(addr)
		if err == nil {
			// connected.
			return conn
		}

		log.Errorf("Could not connect to RabbitMQ. addr: %s, err: %v", addr, err)
		time.Sleep(1 * time.Second)
	}
}

// handleARIEvent handles rabbitMQ connection and message delivery.
func handleARIEvent() {
	for {
		// connect to rabbitmq
		conn := connectRabbitMQ(*rabbitAddr)
		defer conn.Close()

		// create channel
		ch, err := conn.Channel()
		if err != nil {
			log.Errorf("Could not create a channel for RabbitMQ send. err: %v", err)

			time.Sleep(time.Second * 1)
			continue
		}
		defer ch.Close()

		publishMessage()

		<-chRabbitMQReconnect
		time.Sleep(time.Second * 1)
	}
}

// handleARIRequest handles Asterisk request through the rabbit RPC.
func handleARIRequest() {
	// connect new queue.
	q := rabbitmq.NewQueue(*rabbitAddr, *rabbitQueueARIRequest, false)
	q.ConsumeRPC("", rpc.RequestHandler)
}

// consumeMessage publish the Asterisk ARI event to the queue.
func publishMessage() {
	q := rabbitmq.NewQueue(*rabbitAddr, *rabbitQueueARIEvent, true)

	go func(*rabbitmq.Queue) {
		for {
			select {
			case msg := <-chARIEvent:
				event := &rabbitmq.Event{
					Type:     "ari_event",
					DataType: "application/json",
					Data:     string(msg),
				}
				q.PublishMessage(event)
			}
		}
	}(q)
}
