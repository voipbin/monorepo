package subscribehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-schedule-manager/pkg/schedulehandler"
)

const testWait = 5 * time.Second

// Test_Run_failures verifies that a failing broker call stops Run() with a
// wrapped error, and that the consumer is never started afterwards. The strict
// controller fails the test on any call that has no expectation.
func Test_Run_failures(t *testing.T) {
	queueName := string(commonoutline.QueueNameScheduleSubscribe)
	exchange := string(commonoutline.QueueNameEvent)

	tests := []struct {
		name  string
		setup func(m *sockhandler.MockSockHandler)

		expectContains []string
	}{
		{
			name: "queue declare fails",
			setup: func(m *sockhandler.MockSockHandler) {
				m.EXPECT().QueueCreate(queueName, "normal").Return(fmt.Errorf("broker down"))
			},

			expectContains: []string{"could not declare the queue for subscribeHandler", "broker down"},
		},
		{
			name: "topic exchange declare fails",
			setup: func(m *sockhandler.MockSockHandler) {
				m.EXPECT().QueueCreate(queueName, "normal").Return(nil)
				m.EXPECT().TopicCreateWithKind(exchange, "topic").Return(fmt.Errorf("exchange refused"))
			},

			expectContains: []string{"could not declare the global topic exchange", "exchange refused"},
		},
		{
			name: "queue bind fails on the first pattern",
			setup: func(m *sockhandler.MockSockHandler) {
				m.EXPECT().QueueCreate(queueName, "normal").Return(nil)
				m.EXPECT().TopicCreateWithKind(exchange, "topic").Return(nil)
				m.EXPECT().QueueBind(queueName, topicPatterns[0], exchange, false, nil).Return(fmt.Errorf("bind refused"))
			},

			expectContains: []string{"could not bind the topic pattern", topicPatterns[0], "bind refused"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockSchedule := schedulehandler.NewMockScheduleHandler(mc)
			tt.setup(mockSock)

			h := NewSubscribeHandler(mockSock, queueName, mockSchedule)

			err := h.Run()
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			for _, want := range tt.expectContains {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("Wrong match. expect: %q in the error, got: %v", want, err)
				}
			}
		})
	}
}

func Test_processEventRun_is_asynchronous(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSchedule := schedulehandler.NewMockScheduleHandler(mc)

	customer := &cmcustomer.Customer{ID: uuid.FromStringOrNil("29c7a2be-6f23-11f0-b3f6-9f2a3ba4bd12")}
	data, err := json.Marshal(customer)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})

	// the handler stays blocked until the test has seen processEventRun return
	mockSchedule.EXPECT().EventCustomerDeleted(gomock.Any(), customer).
		DoAndReturn(func(_ context.Context, _ *cmcustomer.Customer) error {
			close(started)
			<-release
			close(finished)

			return nil
		})

	h := &subscribeHandler{sockHandler: mockSock, scheduleHandler: mockSchedule}
	event := &sock.Event{
		Publisher: publisherCustomerManager,
		Type:      string(cmcustomer.EventTypeCustomerDeleted),
		Data:      json.RawMessage(data),
	}

	returned := make(chan error, 1)
	go func() {
		returned <- h.processEventRun(event)
	}()

	select {
	case errRun := <-returned:
		if errRun != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", errRun)
		}
	case <-time.After(testWait):
		close(release)
		t.Fatalf("Wrong match. expect: processEventRun returned while the handler is still running, got: timeout")
	}

	select {
	case <-started:
	case <-time.After(testWait):
		close(release)
		t.Fatalf("Wrong match. expect: the event handled in the background, got: timeout")
	}

	close(release)
	select {
	case <-finished:
	case <-time.After(testWait):
		t.Fatalf("Wrong match. expect: the handler completed, got: timeout")
	}
}
