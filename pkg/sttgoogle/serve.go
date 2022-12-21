package sttgoogle

import (
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
)

// serveListen starts the UDP listen.
func (h *streamingHandler) serveListen() (*net.UDPConn, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "serveListen",
		},
	)

	// find available port for 10 times
	for i := 0; i < 10; i++ {
		// get listen port
		port := getRandomPort()
		conn, err := net.ListenUDP("udp", &net.UDPAddr{
			IP:   net.ParseIP(defaultListenIP),
			Port: port,
		})
		if err != nil {
			log.Errorf("Could not listen the address. ip: %s, port: %d, err: %v", defaultListenIP, port, err)
			continue
		}

		return conn, nil
	}

	return nil, fmt.Errorf("no available port")
}
