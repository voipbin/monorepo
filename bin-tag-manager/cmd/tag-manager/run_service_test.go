package main

import (
	"database/sql"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-tag-manager/internal/config"
	"monorepo/bin-tag-manager/pkg/cachehandler"
)

var (
	runServiceOnce   sync.Once
	runServiceBroker *fakeBroker
	runServiceErr    error
)

// startRunService runs runService once per process against the fake broker. The handlers it
// builds register Prometheus collectors, which panics on a second registration, so every test
// run only inspects the shared result.
func startRunService(t *testing.T) {
	t.Helper()

	runServiceOnce.Do(func() {
		broker, err := newFakeBroker()
		if err != nil {
			runServiceErr = err
			return
		}
		runServiceBroker = broker

		saved := *config.Get()
		defer func() { *config.Get() = saved }()
		config.Get().RabbitMQAddress = broker.address

		// the cache handler is never called by the wiring; the controller is not tied to a test
		mc := gomock.NewController(panicReporter{})
		runServiceErr = runService(&sql.DB{}, cachehandler.NewMockCacheHandler(mc))
	})

	if runServiceErr != nil {
		t.Fatalf("runService failed: %v", runServiceErr)
	}
}

// panicReporter fails loudly if the unused cache mock is ever called.
type panicReporter struct{}

func (panicReporter) Errorf(format string, args ...any) { panic(fmt.Sprintf(format, args...)) }
func (panicReporter) Fatalf(format string, args ...any) { panic(fmt.Sprintf(format, args...)) }

func Test_runService(t *testing.T) {
	startRunService(t)

	const (
		requestQueue   = "bin-manager.tag-manager.request"
		subscribeQueue = "bin-manager.tag-manager.subscribe"
		eventExchange  = "bin-manager.event"
		customerDelete = "customer-manager.customer.*.deleted"
	)

	// the consumers are started on goroutines, so wait for the whole expected state
	deadline := time.Now().Add(5 * time.Second)
	var queues, consumers []string
	var binds []declaredBind
	for {
		queues, binds, consumers = runServiceBroker.snapshot()
		if slices.Contains(consumers, requestQueue) && slices.Contains(consumers, subscribeQueue) {
			break
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	for _, q := range []string{requestQueue, subscribeQueue} {
		if !slices.Contains(queues, q) {
			t.Errorf("The queue was not declared. queue: %s, declared: %v", q, queues)
		}
		if !slices.Contains(consumers, q) {
			t.Errorf("The queue has no consumer. queue: %s, consumers: %v", q, consumers)
		}
	}

	// the subscribe queue is bound to the global event exchange with the customer delete pattern
	var gotKeys []string
	for _, b := range binds {
		if b.queue == subscribeQueue && b.exchange == eventExchange {
			gotKeys = append(gotKeys, b.key)
		}
	}
	if !slices.Contains(gotKeys, customerDelete) {
		t.Errorf("The subscribe queue is not bound to the customer delete event. expect key: %s, got keys: %v", customerDelete, gotKeys)
	}
}
