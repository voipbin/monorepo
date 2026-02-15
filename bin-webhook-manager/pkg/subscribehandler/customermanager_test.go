package subscribehandler

import (
	"fmt"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webhook-manager/models/account"
	"monorepo/bin-webhook-manager/pkg/accounthandler"
)

func TestProcessEventCSCustomerCreatedCreated(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)

	h := &subscribeHandler{
		sockHandler:    mockSock,
		accountHandler: mockAccount,
	}

	tests := []struct {
		name    string
		event   *sock.Event
		message *cscustomer.Customer
	}{
		{
			"normal",
			&sock.Event{
				Type:      cscustomer.EventTypeCustomerCreated,
				Publisher: publisherCustomerManager,
				DataType:  "application/json",
				Data:      []byte(`{"id":"c03b033e-8351-11ec-82e6-774ce7627f1b","webhook_method":"POST","webhook_uri":"test.com"}`),
			},
			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("c03b033e-8351-11ec-82e6-774ce7627f1b"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockAccount.EXPECT().UpdateByCustomer(gomock.Any(), tt.message).Return(&account.Account{}, nil)

			h.processEvent(tt.event)

		})
	}
}

func TestProcessEventCSCustomerCreatedUpdated(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)

	h := &subscribeHandler{
		sockHandler:    mockSock,
		accountHandler: mockAccount,
	}

	tests := []struct {
		name    string
		event   *sock.Event
		message *cscustomer.Customer
	}{
		{
			"normal",
			&sock.Event{
				Type:      cscustomer.EventTypeCustomerUpdated,
				Publisher: publisherCustomerManager,
				DataType:  "application/json",
				Data:      []byte(`{"id":"4aca412c-833e-11ec-b806-c7284f1cbb4a","webhook_method":"POST","webhook_uri":"test.com"}`),
			},
			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("4aca412c-833e-11ec-b806-c7284f1cbb4a"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockAccount.EXPECT().UpdateByCustomer(gomock.Any(), tt.message).Return(&account.Account{}, nil)

			h.processEvent(tt.event)

		})
	}
}

func TestProcessEventCSCustomerCreatedError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)

	h := &subscribeHandler{
		sockHandler:    mockSock,
		accountHandler: mockAccount,
	}

	tests := []struct {
		name    string
		event   *sock.Event
		message *cscustomer.Customer
	}{
		{
			"update_error",
			&sock.Event{
				Type:      cscustomer.EventTypeCustomerCreated,
				Publisher: publisherCustomerManager,
				DataType:  "application/json",
				Data:      []byte(`{"id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","webhook_method":"POST","webhook_uri":"test.com"}`),
			},
			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAccount.EXPECT().UpdateByCustomer(gomock.Any(), tt.message).Return(nil, fmt.Errorf("update error"))

			h.processEvent(tt.event)
		})
	}
}

func TestProcessEventCSCustomerInvalidJSON(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)

	h := &subscribeHandler{
		sockHandler:    mockSock,
		accountHandler: mockAccount,
	}

	tests := []struct {
		name  string
		event *sock.Event
	}{
		{
			"invalid_json",
			&sock.Event{
				Type:      cscustomer.EventTypeCustomerCreated,
				Publisher: publisherCustomerManager,
				DataType:  "application/json",
				Data:      []byte(`invalid json`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h.processEvent(tt.event)
		})
	}
}

func TestProcessEventUnknown(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)

	h := &subscribeHandler{
		sockHandler:    mockSock,
		accountHandler: mockAccount,
	}

	tests := []struct {
		name  string
		event *sock.Event
	}{
		{
			"unknown_publisher",
			&sock.Event{
				Type:      "some_event",
				Publisher: "unknown-publisher",
				DataType:  "application/json",
				Data:      []byte(`{}`),
			},
		},
		{
			"unknown_event_type",
			&sock.Event{
				Type:      "unknown_event",
				Publisher: publisherCustomerManager,
				DataType:  "application/json",
				Data:      []byte(`{}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h.processEvent(tt.event)
		})
	}
}

func TestProcessEventRun(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)

	h := &subscribeHandler{
		sockHandler:    mockSock,
		accountHandler: mockAccount,
	}

	event := &sock.Event{
		Type:      "test_event",
		Publisher: "test_publisher",
		DataType:  "application/json",
		Data:      []byte(`{}`),
	}

	err := h.processEventRun(event)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func TestNewSubscribeHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)

	h := NewSubscribeHandler(mockSock, "test-queue", "target1,target2", mockAccount)
	if h == nil {
		t.Errorf("Wrong match. expect: handler, got: nil")
	}
}
