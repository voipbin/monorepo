package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-storage-manager/internal/config"
	"monorepo/bin-storage-manager/pkg/accounthandler"
	"monorepo/bin-storage-manager/pkg/filehandler"
	"monorepo/bin-storage-manager/pkg/storagehandler"
)

var errSimulated = errors.New("simulated failure")

const waitTimeout = 5 * time.Second

func Test_rootCmd_flags(t *testing.T) {
	// The defaults of database_dsn and rabbitmq_address are deployment specific, so only their presence is checked.
	for _, name := range []string{"database_dsn", "rabbitmq_address"} {
		if rootCmd.Flags().Lookup(name) == nil {
			t.Errorf("Wrong match. expect flag: %s", name)
		}
	}

	tests := []struct {
		name   string
		kind   string
		expect string
	}{
		{"prometheus_endpoint", "string", "/metrics"},
		{"prometheus_listen_address", "string", ":2112"},
		{"redis_address", "string", "127.0.0.1:6379"},
		{"redis_database", "int", "1"},
		{"redis_password", "string", ""},
		{"gcp_project_id", "string", ""},
		{"gcp_bucket_name_media", "string", ""},
		{"gcp_bucket_name_tmp", "string", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := rootCmd.Flags().Lookup(tt.name)
			if f == nil {
				t.Fatalf("Wrong match. expect flag: %s", tt.name)
			}
			if f.Value.Type() != tt.kind {
				t.Errorf("Wrong match. expect type: %s, got: %s", tt.kind, f.Value.Type())
			}
			if f.DefValue != tt.expect {
				t.Errorf("Wrong match. expect default: %q, got: %q", tt.expect, f.DefValue)
			}
		})
	}

	if rootCmd.Use != "storage-manager" {
		t.Errorf("Wrong match. expect: storage-manager, got: %s", rootCmd.Use)
	}
	if rootCmd.RunE == nil {
		t.Errorf("Wrong match. expect RunE to be set")
	}
}

func Test_createDBHandler_invalidDSN(t *testing.T) {
	old := *config.Get()
	config.Get().DatabaseDSN = "invalid"
	t.Cleanup(func() { *config.Get() = old })

	res, err := createDBHandler()
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if !strings.Contains(err.Error(), "invalid DSN") {
		t.Errorf("Wrong match. expect error containing: invalid DSN, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil handler, got: %v", res)
	}
}

func Test_runListen(t *testing.T) {
	queue := string(commonoutline.QueueNameStorageRequest)

	t.Run("queue create failed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(errSimulated)

		err := runListen(mockSock, storagehandler.NewMockStorageHandler(mc), accounthandler.NewMockAccountHandler(mc))
		if !errors.Is(err, errSimulated) {
			t.Errorf("Wrong match. expect: %v, got: %v", errSimulated, err)
		}
	})

	t.Run("consumer started", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		started := make(chan string, 1)
		mockSock := sockhandler.NewMockSockHandler(mc)
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, gomock.Any(), false, false, false, 10, gomock.Any()).
			DoAndReturn(func(ctx context.Context, queueName, consumerName string, exclusive, noLocal, noWait bool, workerNum int, cb sock.CbMsgRPC) error {
				started <- queueName
				return nil
			})

		if err := runListen(mockSock, storagehandler.NewMockStorageHandler(mc), accounthandler.NewMockAccountHandler(mc)); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		select {
		case got := <-started:
			if got != queue {
				t.Errorf("Wrong match. expect: %s, got: %s", queue, got)
			}
		case <-time.After(waitTimeout):
			t.Fatalf("Wrong match. the consumer was not started")
		}
	})
}

func Test_runSubscribe(t *testing.T) {
	queue := string(commonoutline.QueueNameStorageSubscribe)

	t.Run("queue create failed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(errSimulated)

		err := runSubscribe(mockSock, queue, accounthandler.NewMockAccountHandler(mc), filehandler.NewMockFileHandler(mc))
		if !errors.Is(err, errSimulated) {
			t.Errorf("Wrong match. expect: %v, got: %v", errSimulated, err)
		}
	})

	t.Run("consumer started", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		started := make(chan string, 1)
		mockSock := sockhandler.NewMockSockHandler(mc)
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil)
		mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
		// the topic patterns are private to the subscribehandler package, so they are matched loosely
		mockSock.EXPECT().QueueBind(queue, gomock.Any(), string(commonoutline.QueueNameEvent), false, nil).Return(nil).AnyTimes()
		mockSock.EXPECT().ConsumeMessage(gomock.Any(), queue, gomock.Any(), false, false, false, 10, gomock.Any()).
			DoAndReturn(func(ctx context.Context, queueName, consumerName string, exclusive, noLocal, noWait bool, numWorkers int, cb sock.CbMsgConsume) error {
				started <- queueName
				return nil
			})

		if err := runSubscribe(mockSock, queue, accounthandler.NewMockAccountHandler(mc), filehandler.NewMockFileHandler(mc)); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		select {
		case got := <-started:
			if got != queue {
				t.Errorf("Wrong match. expect: %s, got: %s", queue, got)
			}
		case <-time.After(waitTimeout):
			t.Fatalf("Wrong match. the consumer was not started")
		}
	})
}
