package main

import (
	"os"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/voip-kamailio-proxy/pkg/listenhandler"
	"monorepo/voip-kamailio-proxy/pkg/siphandler"

	"github.com/sirupsen/logrus"
)

var (
	rabbitMQAddress     = ""
	rabbitMQQueueListen = ""

	prometheusEndpoint      = ""
	prometheusListenAddress = ""

	sipTimeout = 5 * time.Second
)

var chSigs = make(chan os.Signal, 1)

func main() {
	log := logrus.WithField("func", "main")

	log.Debugf("Connecting to RabbitMQ. address: %s", rabbitMQAddress)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	listenHandler := listenhandler.NewListenHandler(
		sockHandler,
		rabbitMQQueueListen,
		sipTimeout,
		siphandler.SIPChecker(siphandler.SendOptionsCheck),
	)

	if err := listenHandler.Run(); err != nil {
		log.Errorf("Could not run the listen handler: %v", err)
		return
	}
	log.Infof("kamailio-proxy is running. queue: %s", rabbitMQQueueListen)

	sig := <-chSigs
	log.Infof("Terminating kamailio-proxy. sig: %v", sig)
}
