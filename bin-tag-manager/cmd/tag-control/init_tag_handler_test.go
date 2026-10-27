package main

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-tag-manager/internal/config"
	"monorepo/bin-tag-manager/pkg/taghandler"
)

// The broker client and the request handler register process-wide collectors and start
// goroutines that live until the process ends, so initTagHandler runs once per process and
// every test run checks the shared result.
var (
	initTagHandlerOnce   sync.Once
	initTagHandlerBroker *fakeBroker
	initTagHandlerDone   = make(chan struct{})
	initTagHandlerResult struct {
		handler taghandler.TagHandler
		err     error
		panicV  any
	}
	initTagHandlerExited atomic.Bool
)

func Test_initTagHandler(t *testing.T) {
	initTagHandlerOnce.Do(func() {
		broker, err := newFakeBroker()
		if err != nil {
			t.Fatalf("could not start the fake broker: %v", err)
		}
		initTagHandlerBroker = broker

		// a Fatalf in the notify handler must not end the test binary
		savedExit := logrus.StandardLogger().ExitFunc
		logrus.StandardLogger().ExitFunc = func(int) { initTagHandlerExited.Store(true) }
		savedAddress := config.Get().RabbitMQAddress
		config.Get().RabbitMQAddress = broker.address
		t.Cleanup(func() {
			logrus.StandardLogger().ExitFunc = savedExit
			config.Get().RabbitMQAddress = savedAddress
		})

		// the goroutine does not touch t, so a late result cannot fail a finished test
		go func() {
			defer close(initTagHandlerDone)
			defer func() { initTagHandlerResult.panicV = recover() }()

			initTagHandlerResult.handler, initTagHandlerResult.err = initTagHandler(nil, nil)
		}()
	})

	select {
	case <-initTagHandlerDone:
	case <-time.After(10 * time.Second):
		t.Fatalf("initTagHandler did not return in time")
	}

	if initTagHandlerResult.panicV != nil {
		t.Fatalf("initTagHandler panicked: %v", initTagHandlerResult.panicV)
	}
	if initTagHandlerExited.Load() {
		t.Errorf("a fatal log was raised while the exchange was declared")
	}
	if initTagHandlerResult.err != nil {
		t.Errorf("Wrong error. expect: nil, got: %v", initTagHandlerResult.err)
	}
	if initTagHandlerResult.handler == nil {
		t.Errorf("Wrong handler. expect: not nil, got: nil")
	}

	// the event stream is published on the global topic exchange
	found := false
	for _, e := range initTagHandlerBroker.declared() {
		if e.name == string(commonoutline.QueueNameEvent) {
			found = true
			if e.kind != "topic" {
				t.Errorf("Wrong exchange kind. exchange: %s, expect: topic, got: %s", e.name, e.kind)
			}
		}
	}
	if !found {
		t.Errorf("The global topic exchange was not declared. declared: %v", initTagHandlerBroker.declared())
	}
}
