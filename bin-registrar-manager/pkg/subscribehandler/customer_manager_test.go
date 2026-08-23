package subscribehandler

import (
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cucustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-registrar-manager/pkg/customerdomainhandler"
	"monorepo/bin-registrar-manager/pkg/extensionhandler"
	"monorepo/bin-registrar-manager/pkg/trunkhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_processEvent_processEventCMCustomerDeleted(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectCustomer *cucustomer.Customer
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: "customer-manager",
				Type:      cucustomer.EventTypeCustomerDeleted,
				DataType:  "application/json",
				Data:      []byte(`{"id":"4f8fbc3c-ccca-11ee-8104-9f5b184cb220"}`),
			},

			expectCustomer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("4f8fbc3c-ccca-11ee-8104-9f5b184cb220"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockExtension := extensionhandler.NewMockExtensionHandler(mc)
			mockTrunk := trunkhandler.NewMockTrunkHandler(mc)
			mockCustomerDomain := customerdomainhandler.NewMockCustomerDomainHandler(mc)

			h := subscribeHandler{
				sockHandler: mockSock,

				extensionHandler:      mockExtension,
				trunkHandler:          mockTrunk,
				customerDomainHandler: mockCustomerDomain,
			}

			mockTrunk.EXPECT().EventCUCustomerDeleted(gomock.Any(), tt.expectCustomer).Return(nil)
			mockExtension.EXPECT().EventCUCustomerDeleted(gomock.Any(), tt.expectCustomer).Return(nil)
			mockCustomerDomain.EXPECT().EventCUCustomerDeleted(gomock.Any(), tt.expectCustomer).Return(nil)

			h.processEvent(tt.event)
		})
	}
}

func Test_processEvent_processEventCMCustomerCreated(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectCustomer *cucustomer.Customer
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: "customer-manager",
				Type:      cucustomer.EventTypeCustomerCreated,
				DataType:  "application/json",
				Data:      []byte(`{"id":"9a1b0c2e-7f7f-11ee-8f5a-1f2e3d4c5b6a"}`),
			},

			expectCustomer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("9a1b0c2e-7f7f-11ee-8f5a-1f2e3d4c5b6a"),
			},
		},
		{
			name: "customer created event carries the headless flag",

			event: &sock.Event{
				Publisher: "customer-manager",
				Type:      cucustomer.EventTypeCustomerCreated,
				DataType:  "application/json",
				Data:      []byte(`{"id":"c0e6f0a2-7f7f-11ee-9d3b-3f2e1d0c9b8a","headless":true}`),
			},

			expectCustomer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("c0e6f0a2-7f7f-11ee-9d3b-3f2e1d0c9b8a"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockExtension := extensionhandler.NewMockExtensionHandler(mc)
			mockTrunk := trunkhandler.NewMockTrunkHandler(mc)
			mockCustomerDomain := customerdomainhandler.NewMockCustomerDomainHandler(mc)

			h := subscribeHandler{
				sockHandler: mockSock,

				extensionHandler:      mockExtension,
				trunkHandler:          mockTrunk,
				customerDomainHandler: mockCustomerDomain,
			}

			mockCustomerDomain.EXPECT().EventCUCustomerCreated(gomock.Any(), tt.expectCustomer).Return(nil)

			h.processEvent(tt.event)
		})
	}
}

func Test_processEvent_ignoresUnhandledEvents(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event
	}{
		{
			name: "customer updated event is ignored",

			event: &sock.Event{
				Publisher: "customer-manager",
				Type:      cucustomer.EventTypeCustomerUpdated,
				DataType:  "application/json",
				Data:      []byte(`{"id":"4f8fbc3c-ccca-11ee-8104-9f5b184cb220"}`),
			},
		},
		{
			name: "customer created from another publisher is ignored",

			event: &sock.Event{
				Publisher: "call-manager",
				Type:      cucustomer.EventTypeCustomerCreated,
				DataType:  "application/json",
				Data:      []byte(`{"id":"4f8fbc3c-ccca-11ee-8104-9f5b184cb220"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockExtension := extensionhandler.NewMockExtensionHandler(mc)
			mockTrunk := trunkhandler.NewMockTrunkHandler(mc)
			mockCustomerDomain := customerdomainhandler.NewMockCustomerDomainHandler(mc)

			h := subscribeHandler{
				sockHandler: mockSock,

				extensionHandler:      mockExtension,
				trunkHandler:          mockTrunk,
				customerDomainHandler: mockCustomerDomain,
			}

			// no handler expectations: any call fails the test
			h.processEvent(tt.event)
		})
	}
}
