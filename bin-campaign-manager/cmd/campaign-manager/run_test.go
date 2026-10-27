package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/internal/config"
	"monorepo/bin-campaign-manager/pkg/campaigncallhandler"
	"monorepo/bin-campaign-manager/pkg/campaignhandler"
	"monorepo/bin-campaign-manager/pkg/outplanhandler"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

// badDSN is rejected while it is parsed, so no connection is attempted.
const badDSN = "%%%bad"

// promPathSeq gives every initProm call of the test binary its own path, because the default
// HTTP mux panics on a duplicate pattern (-count and -shuffle run tests more than once).
var promPathSeq atomic.Int64

func uniquePromPath() string {
	return fmt.Sprintf("/metrics-test-%d", promPathSeq.Add(1))
}

// overrideConfig replaces the fields of the shared, once-loaded config and restores them.
func overrideConfig(t *testing.T, c config.Config) {
	t.Helper()

	shared := config.Load()
	saved := *shared
	*shared = c
	t.Cleanup(func() { *shared = saved })
}

func drainSignalChannels() {
	for {
		select {
		case <-chSigs:
		case <-chDone:
		default:
			return
		}
	}
}

func Test_createDBHandler_invalidDSN(t *testing.T) {
	res, err := createDBHandler(&config.Config{DatabaseDSN: badDSN})
	if err == nil {
		t.Fatal("Wrong match. expect: an error for an invalid database address, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: no handler, got: %v", res)
	}
}

func Test_runService_stopsAtDatabaseFailure(t *testing.T) {
	overrideConfig(t, config.Config{
		DatabaseDSN:             badDSN,
		PrometheusEndpoint:      uniquePromPath(),
		PrometheusListenAddress: "127.0.0.1:0",
	})
	drainSignalChannels()
	t.Cleanup(drainSignalChannels)

	if err := runService(nil, nil); err == nil {
		t.Fatal("Wrong match. expect: the database error, got: nil")
	}

	// the service did not reach the point where it waits for the done signal
	select {
	case <-chDone:
		t.Error("Wrong match. expect: no done signal, got: one")
	default:
	}
}

func Test_initProm_servesMetrics(t *testing.T) {
	// reserve a free loopback port, then give it to the metrics listener
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not reserve a port: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()

	path := uniquePromPath()
	initProm(path, addr)

	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, errGet := client.Get("http://" + addr + path)
		if errGet == nil {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "go_goroutines") {
				t.Fatalf("Wrong match. expect: 200 with the go metrics, got: %d", resp.StatusCode)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("the metrics endpoint did not come up: %v", errGet)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func Test_signalHandler_setsDone(t *testing.T) {
	drainSignalChannels()
	t.Cleanup(drainSignalChannels)

	// the handler started by init consumes one signal per process, so this test runs its own
	// handler. Whichever one receives the signal fills the done channel once.
	go signalHandler()
	chSigs <- syscall.SIGTERM

	select {
	case <-chDone:
	case <-time.After(5 * time.Second):
		t.Fatal("the done signal was not sent")
	}
}

func Test_runListen(t *testing.T) {
	queue := string(commonoutline.QueueNameCampaignRequest)

	t.Run("queue failure does not start the consumer", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		// ConsumeRPC has no expectation, so starting the consumer fails the test
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(errors.New("simulated queue failure"))

		err := runListen(mockSock, outplanhandler.NewMockOutplanHandler(mc), campaignhandler.NewMockCampaignHandler(mc), campaigncallhandler.NewMockCampaigncallHandler(mc))
		if err != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", err)
		}
	})

	t.Run("consumer is started on the request queue", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		started := make(chan struct{})
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, "campaign-manager", false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ any, _, _ string, _, _, _ bool, _ int, _ any) error {
				close(started)
				return errors.New("simulated consume failure")
			})

		err := runListen(mockSock, outplanhandler.NewMockOutplanHandler(mc), campaignhandler.NewMockCampaignHandler(mc), campaigncallhandler.NewMockCampaigncallHandler(mc))
		if err != nil {
			t.Fatalf("Wrong match. expect: nil, got: %v", err)
		}

		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("the consumer was not started")
		}
	})
}

func Test_runSubscribe(t *testing.T) {
	queue := string(commonoutline.QueueNameCampaignSubscribe)

	t.Run("queue failure is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(errors.New("simulated queue failure"))

		err := runSubscribe(mockSock, outplanhandler.NewMockOutplanHandler(mc), campaignhandler.NewMockCampaignHandler(mc), campaigncallhandler.NewMockCampaigncallHandler(mc))
		if err == nil || !strings.Contains(err.Error(), "simulated queue failure") {
			t.Errorf("Wrong match. expect: the queue error, got: %v", err)
		}
	})

	t.Run("consumer is started after the bindings", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		started := make(chan struct{})
		gomock.InOrder(
			mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil),
			mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil),
			mockSock.EXPECT().QueueBind(queue, gomock.Any(), string(commonoutline.QueueNameEvent), false, nil).Return(nil).AnyTimes(),
		)
		mockSock.EXPECT().ConsumeMessage(gomock.Any(), queue, string(commonoutline.ServiceNameCampaignManager), false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ any, _, _ string, _, _, _ bool, _ int, _ any) error {
				close(started)
				return nil
			})

		err := runSubscribe(mockSock, outplanhandler.NewMockOutplanHandler(mc), campaignhandler.NewMockCampaignHandler(mc), campaigncallhandler.NewMockCampaigncallHandler(mc))
		if err != nil {
			t.Fatalf("Wrong match. expect: nil, got: %v", err)
		}

		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("the consumer was not started")
		}
	})
}
