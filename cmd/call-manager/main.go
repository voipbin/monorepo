package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	joonix "github.com/joonix/log"
	log "github.com/sirupsen/logrus"
)

var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

func main() {
	fmt.Println("Hello world!")

	log.WithFields(log.Fields{
		"msg": "hello",
	}).Debug("Hello world!")

	// run workers
	go signalHandler()
	go timeClock()

	<-chDone

	return
}

// proces init
func init() {
	log.SetFormatter(joonix.NewFormatter())
	// log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.DebugLevel)
	// log.SetOutput(os.Stdout)

	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGHUP)
}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	log.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

// timeClock prints current time per each seconds.
func timeClock() {
	log.Debug("timeClock started.")
	for {
		log.Debugf("Current time is: %v", time.Now())
		time.Sleep(time.Second * 1)
	}
}
