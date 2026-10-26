package subscribehandler

import (
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

	"monorepo/bin-tag-manager/pkg/taghandler"
)

func Test_Run_topicCreateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTag := taghandler.NewMockTagHandler(mc)
	h := &subscribeHandler{
		sockHandler:    mockSock,
		subscribeQueue: "test-queue",
		tagHandler:     mockTag,
	}

	// no QueueBind or ConsumeMessage expectation: reaching them fails the test
	mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(fmt.Errorf("exchange failed"))

	err := h.Run()
	if err == nil || !strings.Contains(err.Error(), "could not declare the global topic exchange") {
		t.Errorf("Wrong error. got: %v", err)
	}
}

func Test_Run_queueBindError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTag := taghandler.NewMockTagHandler(mc)
	h := &subscribeHandler{
		sockHandler:    mockSock,
		subscribeQueue: "test-queue",
		tagHandler:     mockTag,
	}

	// no ConsumeMessage expectation: a failed bind must not start consuming
	mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
	mockSock.EXPECT().QueueBind("test-queue", topicPatterns[0], string(commonoutline.QueueNameEvent), false, nil).Return(fmt.Errorf("bind failed"))

	err := h.Run()
	if err == nil || !strings.Contains(err.Error(), "could not bind the topic pattern") || !strings.Contains(err.Error(), topicPatterns[0]) {
		t.Errorf("Wrong error. got: %v", err)
	}
}

func Test_Run_consumeError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTag := taghandler.NewMockTagHandler(mc)
	h := &subscribeHandler{
		sockHandler:    mockSock,
		subscribeQueue: "test-queue",
		tagHandler:     mockTag,
	}

	consumed := make(chan struct{})
	mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
	mockSock.EXPECT().QueueBind("test-queue", topicPatterns[0], string(commonoutline.QueueNameEvent), false, nil).Return(nil)
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), "test-queue", string(commonoutline.ServiceNameTagManager), false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_ any, _, _ string, _, _, _ bool, _ int, _ any) error {
			defer close(consumed)
			return fmt.Errorf("consume failed")
		})

	if err := h.Run(); err != nil {
		t.Fatalf("Run should not return the consume error, got: %v", err)
	}

	select {
	case <-consumed:
	case <-time.After(5 * time.Second):
		t.Fatalf("ConsumeMessage was not called in time")
	}
}

func Test_processEvent_handlerError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTag := taghandler.NewMockTagHandler(mc)
	h := &subscribeHandler{
		sockHandler: mockSock,
		tagHandler:  mockTag,
	}

	customer := &cmcustomer.Customer{
		ID: uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6"),
	}
	data, err := json.Marshal(customer)
	if err != nil {
		t.Fatalf("could not marshal the customer: %v", err)
	}

	// the failure is only logged; processEvent must neither panic nor retry
	mockTag.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).Return(fmt.Errorf("delete failed")).Times(1)

	h.processEvent(&sock.Event{
		Publisher: publisherCustomerManager,
		Type:      string(cmcustomer.EventTypeCustomerDeleted),
		Data:      json.RawMessage(data),
	})
}
