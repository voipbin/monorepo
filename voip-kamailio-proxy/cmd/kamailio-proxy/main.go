package main

import (
	"fmt"
	"net"
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

	interfaceName = ""

	prometheusEndpoint      = ""
	prometheusListenAddress = ""

	sipTimeout = 5 * time.Second
)

var chSigs = make(chan os.Signal, 1)

func main() {
	log := logrus.WithField("func", "main")

	kamailioID, err := getKamailioID(interfaceName)
	if err != nil {
		log.Errorf("Could not get kamailio ID from interface %q: %v", interfaceName, err)
		return
	}

	rabbitQueueListenPermanent := rabbitMQQueueListen
	rabbitQueueListenVolatile := fmt.Sprintf("voip.kamailio.%s.request", kamailioID)
	log.Debugf("Volatile listen queue name: %s", rabbitQueueListenVolatile)

	log.Debugf("Connecting to RabbitMQ. address: %s", rabbitMQAddress)
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	listenHandler := listenhandler.NewListenHandler(
		sockHandler,
		rabbitQueueListenPermanent,
		rabbitQueueListenVolatile,
		sipTimeout,
		siphandler.SIPChecker(siphandler.SendOptionsCheck),
	)

	if err := listenHandler.Run(); err != nil {
		log.Errorf("Could not run the listen handler: %v", err)
		return
	}
	log.Infof("kamailio-proxy is running. permanent_queue: %s, volatile_queue: %s", rabbitQueueListenPermanent, rabbitQueueListenVolatile)

	sig := <-chSigs
	log.Infof("Terminating kamailio-proxy. sig: %v", sig)
}

// getKamailioID returns the MAC address of the given network interface,
// used as the per-instance kamailio identity for the volatile queue name.
func getKamailioID(ifaceName string) (string, error) {
	logrus.Debugf("Checking interface name. iface: %s", ifaceName)

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("could not get network interfaces: %w", err)
	}

	for _, iface := range ifaces {
		if iface.Name != ifaceName {
			continue
		}

		mac := iface.HardwareAddr.String()
		if mac == "" {
			return "", fmt.Errorf("interface %q has no MAC address", ifaceName)
		}

		logrus.Debugf("Found kamailio ID. iface: %s, mac: %s", ifaceName, mac)
		return mac, nil
	}

	return "", fmt.Errorf("interface %q not found", ifaceName)
}
