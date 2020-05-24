package main

import (
	"context"
	"flag"
	"fmt"
	"log/syslog"
	"net/url"
	"strings"
	"time"

	"gitlab.com/voipbin/voip/asterisk-proxy/internal/rpc"
	"gitlab.com/voipbin/voip/asterisk-proxy/pkg/rabbitmq"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	lSyslog "github.com/sirupsen/logrus/hooks/syslog"
)

var asteriskID = flag.String("asterisk_id", "00:11:22:33:44:55", "The asterisk id")
var asteriskAddressInternal = flag.String("asterisk_address_internal", "127.0.0.1:5060", "The asterisk internal ip address")

var ariAddr = flag.String("ari_addr", "localhost:8088", "The asterisk-proxy connects to this asterisk ari service address")
var ariAccount = flag.String("ari_account", "asterisk:asterisk", "The asterisk-proxy uses this asterisk ari account info. id:password")
var ariSubscribeAll = flag.String("ari_subscribe_all", "true", "The asterisk-proxy uses this asterisk subscribe all option.")
var ariApplication = flag.String("ari_application", "voipbin", "The asterisk-proxy uses this asterisk ari application name.")

var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "The asterisk-proxy connect to rabbitmq address.")
var rabbitQueueListenRequest = flag.String("rabbit_queue_listen", "asterisk.<asterisk_id>.request,asterisk.call.request", "Comma separated asterisk-proxy's listen request queue name.")
var rabbitQueuePublishEvent = flag.String("rabbit_queue_publish", "asterisk.all.event", "The asterisk-proxy sends the ARI event to this rabbitmq queue name. The queue must be created before.")

var redisAddr = flag.String("redis_addr", "localhost:6379", "The redis address for data caching")
var redisDB = flag.Int("redis_db", 0, "The redis database for caching")

// create message buffer
var chARIEvent = make(chan []byte, 1024000)

func main() {
	initProcess()

	// connect to rabbitmq
	logrus.Debugf("test: %s", *rabbitAddr)
	rabbitSock := rabbitmq.NewRabbit(*rabbitAddr)
	rabbitSock.Connect()

	// handle events from the asterisk
	if err := handleEvent(rabbitSock); err != nil {
		logrus.Errorf("Could not initiate the event handler. err: %v", err)
		return
	}

	// handle request from the request queue
	if err := handleRequest(rabbitSock); err != nil {
		logrus.Errorf("Could not initiate the request handler. err: %v", err)
		return
	}

	if err := handleProxyInfo(*redisAddr, *redisDB, *asteriskID, *asteriskAddressInternal); err != nil {
		logrus.Errorf("Could not initiate proxy info handler. err: %v", err)
		return
	}

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
	}

	// rabbitmq.Initiate()
	rpc.Initiate(*ariAddr, *ariAccount)

	log.Info("asterisk-proxy has initiated.")
}

func handleEvent(rabbitSock rabbitmq.Rabbit) error {
	// asterisk ari message receiver
	go recevieARIEvent()

	// push the message into rabbitmq
	go handleARIEvent(rabbitSock)

	return nil

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

// handleARIEvent handles rabbitMQ connection and message delivery.
func handleARIEvent(rabbitSock rabbitmq.Rabbit) {
	for {
		select {
		case msg := <-chARIEvent:
			event := &rabbitmq.Event{
				Type:     "ari_event",
				DataType: "application/json",
				Data:     string(msg),
			}
			go rabbitSock.PublishEvent(*rabbitQueuePublishEvent, event)
		}
	}
}

// handleARIRequest handles Asterisk request through the rabbit RPC.
func handleRequest(rabbitSock rabbitmq.Rabbit) error {
	queues := strings.Split(*rabbitQueueListenRequest, ",")
	for _, queue := range queues {
		log.Debugf("Declaring request queue. queue: %s, sock: %v", queue, rabbitSock)
		if err := rabbitSock.QueueDeclare(queue, false, false, false, false); err != nil {
			logrus.WithFields(
				logrus.Fields{
					"queue": queue,
				}).Errorf("Could not declare the queue. err: %v", err)
			return err
		}

		go func(sock rabbitmq.Rabbit, name string) {
			for {
				sock.ConsumeRPC(name, "", rpc.RequestHandler)
				time.Sleep(time.Second * 1)
			}
		}(rabbitSock, queue)

	}
	return nil
}

// handleProxyInfo updates the ipaddress for every 3 min
func handleProxyInfo(addr string, db int, id, internalAddress string) error {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       db,
	})

	// update internal address
	key := fmt.Sprintf("asterisk.%s.address-internal", id)
	go func() {
		for {
			client.Set(context.Background(), key, internalAddress, time.Hour*24)
			time.Sleep(time.Minute * 5)
		}
	}()

	return nil
}
