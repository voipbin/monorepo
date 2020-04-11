package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/voipbin/bin-manager/call-manager/internal/arihandler"
	call "gitlab.com/voipbin/bin-manager/call-manager/internal/call"

	joonix "github.com/joonix/log"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

// args for rabbitmq
var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")
var rabbitQueueARIEvent = flag.String("rabbit_queue_arievent", "asterisk_ari_event", "rabbitmq asterisk ari event queue name.")
var rabbitQueueARIRequest = flag.String("rabbit_queue_arirequest", "asterisk_ari_request", "rabbitmq asterisk ari request queue prefix.")

// args for prometheus
var promEndpoint = flag.String("prom_endpoint", "/metrics", "endpoint for prometheus metric collecting.")
var promListenAddr = flag.String("prom_listen_addr", ":2112", "endpoint for prometheus metric collecting.")

func main() {

	// ari event handler
	go arihandler.ReceiveEventQueue(*rabbitAddr, *rabbitQueueARIEvent, "call-manager")

	simple := call.Call{
		ID: uuid.NewV4(),
	}
	fmt.Println(simple)

	<-chDone

	return
}

// proces init
func init() {
	flag.Parse()

	// init logs
	initLog()

	// init signal handler
	initSignal()

	// init prometheus setting
	initProm(*promEndpoint, *promListenAddr)

	log.Info("init finished.")
}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	log.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

// initLog inits log settings.
func initLog() {
	log.SetFormatter(joonix.NewFormatter())
	log.SetLevel(log.DebugLevel)
}

// initSignal inits sinal settings.
func initSignal() {
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGHUP)
	go signalHandler()
}

// initProm inits prometheus settings
func initProm(endpoint, listen string) {
	http.Handle(endpoint, promhttp.Handler())
	go func() {
		for {
			err := http.ListenAndServe(listen, nil)
			if err != nil {
				log.Errorf("Could not start prometheus listener")
				continue
			}
			break
		}
	}()
}
