package main

import (
	"context"
	"flag"
	"fmt"
	"log/syslog"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/ivahaev/amigo"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	lSyslog "github.com/sirupsen/logrus/hooks/syslog"

	"gitlab.com/voipbin/voip/asterisk-proxy/pkg/eventhandler"
	"gitlab.com/voipbin/voip/asterisk-proxy/pkg/listenhandler"
	"gitlab.com/voipbin/voip/asterisk-proxy/pkg/rabbitmq"
)

var asteriskID = flag.String("asterisk_id", "00:11:22:33:44:55", "The asterisk id")
var asteriskAddressInternal = flag.String("asterisk_address_internal", "127.0.0.1:5060", "The asterisk internal ip address")

var ariAddr = flag.String("ari_addr", "localhost:8088", "The asterisk-proxy connects to this asterisk ari service address")
var ariAccount = flag.String("ari_account", "asterisk:asterisk", "The asterisk-proxy uses this asterisk ari account info. id:password")
var ariSubscribeAll = flag.String("ari_subscribe_all", "true", "The asterisk-proxy uses this asterisk subscribe all option.")
var ariApplication = flag.String("ari_application", "voipbin", "The asterisk-proxy uses this asterisk ari application name.")

var amiHost = flag.String("ami_host", "127.0.0.1", "The host address for AMI connection.")
var amiPort = flag.String("ami_port", "5038", "The port number for AMI connection.")
var amiUsername = flag.String("ami_username", "asterisk", "The username for AMI login.")
var amiPassword = flag.String("ami_password", "asterisk", "The password for AMI login.")
var amiEventFilter = flag.String("ami_event_filter", "", "The list of messages for listen.")

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

	// connect to ami
	amiSock := connectAMI(*amiHost, *amiPort, *amiUsername, *amiPassword)

	// create a filter
	amiFilter := strings.Split(*amiEventFilter, ",")

	// create event handler
	evtHandler := eventhandler.NewEventHandler(
		rabbitSock, *rabbitQueueListenRequest, *rabbitQueuePublishEvent,
		*ariAddr, *ariAccount, *ariSubscribeAll, *ariApplication,
		amiSock, amiFilter,
	)

	// run event handler
	if err := evtHandler.Run(); err != nil {
		logrus.Errorf("Could not run the eventhandler correctly. err: %v", err)
		return
	}
	logrus.Debug("The event handler is running now.")

	// create a listen handler
	listenHandler := listenhandler.NewListenHandler(
		rabbitSock, *rabbitQueueListenRequest,
		*ariAddr, *ariAccount,
		amiSock,
	)

	// run listen handler
	if err := listenHandler.Run(); err != nil {
		logrus.Errorf("Could not run the listen handler correctly. err: %v", err)
		return
	}
	logrus.Debug("The listen handler is running now.")

	if err := handleProxyInfo(*redisAddr, *redisDB, *asteriskID, *asteriskAddressInternal); err != nil {
		logrus.Errorf("Could not initiate proxy info handler. err: %v", err)
		return
	}

	forever := make(chan bool)
	<-forever
}

// initProcess inits process
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

	log.Info("asterisk-proxy has initiated.")
}

// connectAMI access and login to the Asterisk's AMI.
func connectAMI(host, port, username, password string) *amigo.Amigo {
	// connect to ami
	settings := &amigo.Settings{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	}

	// create a sock and connect
	amiSock := amigo.New(settings)
	amiSock.Connect()

	// set error log
	amiSock.On("error", func(message string) {
		logrus.Errorf("AMI error: %s", message)
	})

	return amiSock
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
