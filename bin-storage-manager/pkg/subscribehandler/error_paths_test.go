package subscribehandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"monorepo/bin-storage-manager/pkg/accounthandler"
	"monorepo/bin-storage-manager/pkg/filehandler"
)

var errTest = errors.New("simulated failure")

func Test_Run_failures(t *testing.T) {
	queueName := string(commonoutline.QueueNameStorageSubscribe)
	topicName := string(commonoutline.QueueNameEvent)

	if len(topicPatterns) < 2 {
		t.Fatalf("Wrong match. expect: at least 2 topic patterns, got: %d", len(topicPatterns))
	}

	tests := []struct {
		name   string
		expect func(m *sockhandler.MockSockHandler)
	}{
		{
			name: "queue create failed",
			expect: func(m *sockhandler.MockSockHandler) {
				m.EXPECT().QueueCreate(queueName, "normal").Return(errTest)
			},
		},
		{
			name: "topic create failed",
			expect: func(m *sockhandler.MockSockHandler) {
				gomock.InOrder(
					m.EXPECT().QueueCreate(queueName, "normal").Return(nil),
					m.EXPECT().TopicCreateWithKind(topicName, "topic").Return(errTest),
				)
			},
		},
		{
			name: "first bind failed",
			expect: func(m *sockhandler.MockSockHandler) {
				gomock.InOrder(
					m.EXPECT().QueueCreate(queueName, "normal").Return(nil),
					m.EXPECT().TopicCreateWithKind(topicName, "topic").Return(nil),
					m.EXPECT().QueueBind(queueName, topicPatterns[0], topicName, false, nil).Return(errTest),
				)
			},
		},
		{
			name: "second bind failed",
			expect: func(m *sockhandler.MockSockHandler) {
				gomock.InOrder(
					m.EXPECT().QueueCreate(queueName, "normal").Return(nil),
					m.EXPECT().TopicCreateWithKind(topicName, "topic").Return(nil),
					m.EXPECT().QueueBind(queueName, topicPatterns[0], topicName, false, nil).Return(nil),
					m.EXPECT().QueueBind(queueName, topicPatterns[1], topicName, false, nil).Return(errTest),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			tt.expect(mockSock) // ConsumeMessage has no expectation: it must not start after a failure.

			h := NewSubscribeHandler(mockSock, queueName, nil, nil)
			if err := h.Run(); !errors.Is(err, errTest) {
				t.Errorf("Wrong match. expect: %v, got: %v", errTest, err)
			}
		})
	}
}

func Test_processEvent_unknownEvent(t *testing.T) {
	tests := []struct {
		name  string
		event *sock.Event
	}{
		{
			name:  "unknown publisher",
			event: &sock.Event{Publisher: "other-manager", Type: cmcustomer.EventTypeCustomerCreated, Data: []byte(`{}`)},
		},
		{
			name:  "known publisher with an unhandled type",
			event: &sock.Event{Publisher: string(commonoutline.ServiceNameCustomerManager), Type: "customer_updated", Data: []byte(`{}`)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// the mocks have no expectations: any handler call fails the test.
			h := &subscribeHandler{
				sockHandler:    sockhandler.NewMockSockHandler(mc),
				accountHandler: accounthandler.NewMockAccountHandler(mc),
				fileHandler:    filehandler.NewMockFileHandler(mc),
			}

			h.processEvent(tt.event)
		})
	}
}

func Test_processEventRun(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockAccount := accounthandler.NewMockAccountHandler(mc)
	h := &subscribeHandler{accountHandler: mockAccount}

	done := make(chan struct{})
	mockAccount.EXPECT().EventCustomerCreated(gomock.Any(), &cmcustomer.Customer{ID: uuid.FromStringOrNil("17ce05c9-0000-0000-0000-000000000001")}).
		DoAndReturn(func(_ context.Context, _ *cmcustomer.Customer) error {
			close(done)
			return nil
		})

	err := h.processEventRun(&sock.Event{
		Publisher: string(commonoutline.ServiceNameCustomerManager),
		Type:      cmcustomer.EventTypeCustomerCreated,
		Data:      []byte(`{"id":"17ce05c9-0000-0000-0000-000000000001"}`),
	})
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("Wrong match. the event was not processed in the background")
	}
}

func Test_processEventCMCustomerCreated_errors(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("28df16da-0000-0000-0000-000000000001")

	t.Run("invalid json does not reach the handler", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h := &subscribeHandler{accountHandler: accounthandler.NewMockAccountHandler(mc)}
		if err := h.processEventCMCustomerCreated(ctx, &sock.Event{Data: []byte(`{not json`)}); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("account handler failure is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockAccount := accounthandler.NewMockAccountHandler(mc)
		h := &subscribeHandler{accountHandler: mockAccount}
		mockAccount.EXPECT().EventCustomerCreated(ctx, &cmcustomer.Customer{ID: customerID}).Return(errTest)

		err := h.processEventCMCustomerCreated(ctx, &sock.Event{Data: []byte(`{"id":"28df16da-0000-0000-0000-000000000001"}`)})
		if !errors.Is(err, errTest) {
			t.Errorf("Wrong match. expect: %v, got: %v", errTest, err)
		}
	})
}

func Test_processEventCMCustomerDeleted_errors(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("39e027eb-0000-0000-0000-000000000001")
	event := &sock.Event{Data: []byte(`{"id":"39e027eb-0000-0000-0000-000000000001"}`)}
	cu := &cmcustomer.Customer{ID: customerID}

	t.Run("invalid json does not reach any handler", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h := &subscribeHandler{
			accountHandler: accounthandler.NewMockAccountHandler(mc),
			fileHandler:    filehandler.NewMockFileHandler(mc),
		}
		if err := h.processEventCMCustomerDeleted(ctx, &sock.Event{Data: []byte(`{not json`)}); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("a file handler failure stops before the account is deleted", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockAccount := accounthandler.NewMockAccountHandler(mc) // no expectation: the account must survive
		mockFile := filehandler.NewMockFileHandler(mc)
		h := &subscribeHandler{accountHandler: mockAccount, fileHandler: mockFile}
		mockFile.EXPECT().EventCustomerDeleted(ctx, cu).Return(errTest)

		if err := h.processEventCMCustomerDeleted(ctx, event); !errors.Is(err, errTest) {
			t.Errorf("Wrong match. expect: %v, got: %v", errTest, err)
		}
	})

	t.Run("an account handler failure is returned after the files were handled", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockAccount := accounthandler.NewMockAccountHandler(mc)
		mockFile := filehandler.NewMockFileHandler(mc)
		h := &subscribeHandler{accountHandler: mockAccount, fileHandler: mockFile}
		gomock.InOrder(
			mockFile.EXPECT().EventCustomerDeleted(ctx, cu).Return(nil),
			mockAccount.EXPECT().EventCustomerDeleted(ctx, cu).Return(errTest),
		)

		if err := h.processEventCMCustomerDeleted(ctx, event); !errors.Is(err, errTest) {
			t.Errorf("Wrong match. expect: %v, got: %v", errTest, err)
		}
	})
}
