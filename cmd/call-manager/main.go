package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/voipbin/bin-manager/call-manager/internal/arihandler"
	call "gitlab.com/voipbin/bin-manager/call-manager/internal/call"

	joonix "github.com/joonix/log"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")
var rabbitQueueARIEvent = flag.String("rabbit_queue_arievent", "asterisk_ari_event", "rabbitmq asterisk ari event queue name.")
var rabbitQueueARIRequest = flag.String("rabbit_queue_arirequest", "asterisk_ari_request", "rabbitmq asterisk ari request queue prefix.")

func main() {

	// signal handler
	go signalHandler()

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
	log.SetFormatter(joonix.NewFormatter())
	log.SetLevel(log.DebugLevel)

	// init signal handler
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGHUP)
	log.Info("init finished")
}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	log.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}
