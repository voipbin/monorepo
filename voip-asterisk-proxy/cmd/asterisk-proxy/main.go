package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/go-redis/redis/v8"
	"github.com/ivahaev/amigo"
	"github.com/sirupsen/logrus"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/voip-asterisk-proxy/pkg/eventhandler"
	"monorepo/voip-asterisk-proxy/pkg/listenhandler"
)

const (
	serviceName = "asterisk-proxy"
)

var (
	ariAddr         = ""
	ariAccount      = ""
	ariSubscribeAll = ""
	ariApplication  = ""

	amiHost        = ""
	amiPort        = ""
	amiUsername    = ""
	amiPassword    = ""
	amiEventFilter = ""

	interfaceName = ""

	rabbitMQAddress     = ""
	rabbitMQQueueListen = ""

	redisAddr = ""
	redisDB   = 0

	prometheusEndpoint      = ""
	prometheusListenAddress = ""
)

var chSigs = make(chan os.Signal, 1)

func main() {
	log := logrus.WithField("func", "main")

	// connect to rabbitmq
	log.Debugf("rabbitmq address: %s", rabbitMQAddress)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	// connect to ami
	amiSock := connectAMI(amiHost, amiPort, amiUsername, amiPassword)

	// create a filter
	amiFilter := strings.Split(amiEventFilter, ",")

	// get asterisk id and internal address
	asteriskID, asteriskAddress, err := getAsteriskIDAddress(interfaceName)
	if err != nil {
		log.Errorf("Could not get correct asterisk-id, asterisk-address info.")
		return
	}

	// create rabbitmq listen requet queue names
	rabbitQueueListenRequestsPermanent := rabbitMQQueueListen
	rabbitQueueListenRequestsVolatile := fmt.Sprintf("asterisk.%s.request", asteriskID)
	log.Debugf("Volatile listen queue name: %s", rabbitQueueListenRequestsVolatile)

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameAsteriskEventAll, serviceName)

	// create event handler
	evtHandler := eventhandler.NewEventHandler(
		notifyHandler,
		sockHandler,
		string(commonoutline.QueueNameAsteriskEventAll),
		ariAddr,
		ariAccount,
		ariSubscribeAll,
		ariApplication,
		amiSock,
		amiFilter,
	)

	// run event handler
	if err := evtHandler.Run(); err != nil {
		log.Errorf("Could not run the eventhandler correctly. err: %v", err)
		return
	}
	log.Debugf("The event handler is running now. listen: %s", rabbitQueueListenRequestsPermanent)

	// create a listen handler
	listenHandler := listenhandler.NewListenHandler(
		sockHandler,
		rabbitQueueListenRequestsPermanent,
		rabbitQueueListenRequestsVolatile,
		ariAddr,
		ariAccount,
		amiSock,
	)

	// run listen handler
	if err := listenHandler.Run(); err != nil {
		log.Errorf("Could not run the listen handler correctly. err: %v", err)
		return
	}
	log.Debugf("The listen handler is running now. id: %s, address: %v", asteriskID, asteriskAddress)

	if err := handleProxyInfo(redisAddr, redisDB, asteriskID, asteriskAddress); err != nil {
		log.Errorf("Could not initiate proxy info handler. err: %v", err)
		return
	}

	sig := <-chSigs
	log.Infof("Terminating api-manager. sig: %v", sig)

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
