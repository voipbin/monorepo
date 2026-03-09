package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/voip-rtpengine-proxy/pkg/gcsuploader"
	"monorepo/voip-rtpengine-proxy/pkg/listenhandler"
	"monorepo/voip-rtpengine-proxy/pkg/ngclient"
	"monorepo/voip-rtpengine-proxy/pkg/pcapwatcher"
)

const serviceName = "rtpengine-proxy"

var (
	interfaceName    = ""
	rtpengineNGAddress = ""
	rtpengineNGTimeout = ""

	rabbitMQAddress     = ""
	rabbitMQQueueListen = ""

	redisAddress  = ""
	redisPassword = ""
	redisDatabase = 0

	prometheusEndpoint      = ""
	prometheusListenAddress = ""

	recordingDir  = ""
	gcsBucketName = ""
)

var chSigs = make(chan os.Signal, 1)

func main() {
	log := logrus.WithField("func", "main")

	proxyID, proxyAddress, err := getInterfaceIP(interfaceName)
	if err != nil {
		log.Errorf("Could not get proxy ID from interface %s: %v", interfaceName, err)
		return
	}
	log.Infof("Proxy ID: %s, Address: %s", proxyID, proxyAddress)

	if err := registerProxyRedis(redisAddress, redisPassword, redisDatabase, proxyID, proxyAddress); err != nil {
		log.Errorf("Could not register proxy in Redis: %v", err)
		return
	}

	permanentQueue := rabbitMQQueueListen
	volatileQueue := fmt.Sprintf("rtpengine.%s.request", proxyID)
	log.Infof("Permanent queue: %s, Volatile queue: %s", permanentQueue, volatileQueue)

	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	ngTimeout, err := time.ParseDuration(rtpengineNGTimeout)
	if err != nil {
		log.Warnf("Invalid NG timeout %q, using 5s", rtpengineNGTimeout)
		ngTimeout = 5 * time.Second
	}

	ng, err := ngclient.New(rtpengineNGAddress, ngTimeout)
	if err != nil {
		log.Errorf("Could not create NG client: %v", err)
		return
	}
	defer ng.Close()

	lh := listenhandler.NewListenHandler(sockHandler, permanentQueue, volatileQueue, ng)
	if err := lh.Run(); err != nil {
		log.Errorf("Could not run listen handler: %v", err)
		return
	}
	log.Infof("%s running. ID: %s", serviceName, proxyID)

	// Start pcap watcher if recording and GCS are configured
	if recordingDir != "" && gcsBucketName != "" {
		uploader, err := gcsuploader.New(gcsBucketName)
		if err != nil {
			log.WithError(err).Error("could not create GCS uploader, pcap watcher disabled")
		} else {
			defer uploader.Close()

			w := pcapwatcher.New(recordingDir, uploader)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				if err := w.Run(ctx); err != nil {
					log.WithError(err).Error("pcap watcher error")
				}
			}()
			log.WithFields(logrus.Fields{
				"recording_dir": recordingDir,
				"gcs_bucket":    gcsBucketName,
			}).Info("pcap watcher enabled")
		}
	} else {
		log.Info("pcap watcher disabled (RTPENGINE_RECORDING_DIR or GCS_BUCKET_NAME not set)")
	}

	sig := <-chSigs
	log.Infof("Terminating %s. sig: %v", serviceName, sig)
}

// getInterfaceIP returns the IPv4 address of the given network interface.
func getInterfaceIP(ifaceName string) (id string, addr string, err error) {
	ifaces, err := net.Interfaces()
	if err != nil {
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
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
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
				continue
			}
			return ip.String(), ip.String(), nil
		}
	}
	return "", "", fmt.Errorf("no IPv4 address on interface %q", ifaceName)
}

// registerProxyRedis stores the proxy IP in Redis with periodic refresh.
func registerProxyRedis(addr, password string, db int, proxyID, internalAddr string) error {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	key := fmt.Sprintf("rtpengine.%s.address-internal", proxyID)

	// Write synchronously first so the key exists before we start accepting commands.
	if err := client.Set(context.Background(), key, internalAddr, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("initial Redis registration: %w", err)
	}

	go func() {
		for {
			time.Sleep(5 * time.Minute)
			if err := client.Set(context.Background(), key, internalAddr, 24*time.Hour).Err(); err != nil {
				logrus.WithError(err).Errorf("Could not refresh proxy in Redis. key: %s", key)
			}
		}
	}()
	return nil
}
