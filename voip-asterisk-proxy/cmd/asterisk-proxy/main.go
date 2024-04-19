package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"strings"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/go-redis/redis/v8"
	"github.com/ivahaev/amigo"
	joonix "github.com/joonix/log"
	"github.com/sirupsen/logrus"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/voip-asterisk-proxy/pkg/eventhandler"
	"monorepo/voip-asterisk-proxy/pkg/listenhandler"
)

const serviceName = "asterisk-proxy"

var flagARIAddr = flag.String("ari_addr", "localhost:8088", "The asterisk-proxy connects to this asterisk ari service address")
var flagARIAccount = flag.String("ari_account", "asterisk:asterisk", "The asterisk-proxy uses this asterisk ari account info. id:password")
var flagARISubscribeAll = flag.String("ari_subscribe_all", "true", "The asterisk-proxy uses this asterisk subscribe all option.")
var flagARIApplication = flag.String("ari_application", "voipbin", "The asterisk-proxy uses this asterisk ari application name.")

var flagAMIHost = flag.String("ami_host", "127.0.0.1", "The host address for AMI connection.")
var flagAMIPort = flag.String("ami_port", "5038", "The port number for AMI connection.")
var flagAMIUsername = flag.String("ami_username", "asterisk", "The username for AMI login.")
var flagAMIPassword = flag.String("ami_password", "asterisk", "The password for AMI login.")
var flagAMIEventFilter = flag.String("ami_event_filter", "", "The list of messages for listen.")

var flagInterfaceName = flag.String("interface_name", "eth0", "The main interface device name.")

var flagRabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "The asterisk-proxy connect to rabbitmq address.")
var flagRabbitQueueListenRequest = flag.String("rabbit_queue_listen", "asterisk.call.request", "Additional comma separated asterisk-proxy's listen request queue name.")
var flagRabbitQueuePublishEvent = flag.String("rabbit_queue_publish", "asterisk.all.event", "The asterisk-proxy sends the ARI event to this rabbitmq queue name. The queue must be created before.")

var flagRedisAddr = flag.String("redis_addr", "localhost:6379", "The redis address for data caching")
var flagRedisDB = flag.Int("redis_db", 0, "The redis database for caching")

func main() {
	initProcess()

	// connect to rabbitmq
	logrus.Debugf("rabbitmq address: %s", *flagRabbitAddr)
	rabbitSock := rabbitmqhandler.NewRabbit(*flagRabbitAddr)
	rabbitSock.Connect()

	// connect to ami
	amiSock := connectAMI(*flagAMIHost, *flagAMIPort, *flagAMIUsername, *flagAMIPassword)

	// create a filter
	amiFilter := strings.Split(*flagAMIEventFilter, ",")

	// get asterisk id and internal address
	asteriskID, asteriskAddress, err := getAsteriskIDAddress(*flagInterfaceName)
	if err != nil {
		logrus.Errorf("Could not get correct asterisk-id, asterisk-address info.")
		return
	}

	// create rabbitmq listen requet queue names
	rabbitQueueListenRequestsPermanent := *flagRabbitQueueListenRequest
	rabbitQueueListenRequestsVolatile := fmt.Sprintf("asterisk.%s.request", asteriskID)
	logrus.Debugf("Volatile listen queue name: %s", rabbitQueueListenRequestsVolatile)

	reqHandler := requesthandler.NewRequestHandler(rabbitSock, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(rabbitSock, reqHandler, commonoutline.QueueNameAsteriskEventAll, serviceName)

	// create event handler
	evtHandler := eventhandler.NewEventHandler(
		notifyHandler,
		rabbitSock,
		*flagRabbitQueuePublishEvent,
		*flagARIAddr,
		*flagARIAccount,
		*flagARISubscribeAll,
		*flagARIApplication,
		amiSock,
		amiFilter,
	)

	// run event handler
	if err := evtHandler.Run(); err != nil {
		logrus.Errorf("Could not run the eventhandler correctly. err: %v", err)
		return
	}
	logrus.Debugf("The event handler is running now. listen: %s", rabbitQueueListenRequestsPermanent)

	// create a listen handler
	listenHandler := listenhandler.NewListenHandler(
		rabbitSock,
		rabbitQueueListenRequestsPermanent,
		rabbitQueueListenRequestsVolatile,
		*flagARIAddr,
		*flagARIAccount,
		amiSock,
	)

	// run listen handler
	if err := listenHandler.Run(); err != nil {
		logrus.Errorf("Could not run the listen handler correctly. err: %v", err)
		return
	}
	logrus.Debugf("The listen handler is running now. id: %s, address: %v", asteriskID, asteriskAddress)

	if err := handleProxyInfo(*flagRedisAddr, *flagRedisDB, asteriskID, asteriskAddress); err != nil {
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
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)

	logrus.Info("asterisk-proxy has initiated.")
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

// getAsteriskIDAddress returns given interface's mac address, ip address.
func getAsteriskIDAddress(ifaceName string) (string, string, error) {
	logrus.Debugf("Checking interface name. iface: %s", ifaceName)

	ifaces, err := net.Interfaces()
	if err != nil {
		logrus.Errorf("Could not get interface information. err: %v", err)
		return "", "", err
	}

	for _, iface := range ifaces {
		if iface.Name != ifaceName {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return "", "", err
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}

			return iface.HardwareAddr.String(), ip.String(), nil
		}

	}
	logrus.Errorf("Could not find correct interface information.")
	return "", "", fmt.Errorf("no interface found")
}
