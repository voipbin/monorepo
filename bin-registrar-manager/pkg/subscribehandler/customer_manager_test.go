package subscribehandler

import (
	"context"
	"errors"
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

func Test_processEventCMCustomerDeleted_errors(t *testing.T) {
	errFail := errors.New("handler failed")
	data := []byte(`{"id":"4f8fbc3c-ccca-11ee-8104-9f5b184cb220"}`)
	cu := &cucustomer.Customer{ID: uuid.FromStringOrNil("4f8fbc3c-ccca-11ee-8104-9f5b184cb220")}

	tests := []struct {
		name string
		data []byte

		trunkErr  error
		extErr    error
		domainErr error

		// which handlers are expected to be called
		callTrunk  bool
		callExt    bool
		callDomain bool
		wantErr    bool
	}{
		{name: "invalid json", data: []byte(`{"id":`), wantErr: true},
		{name: "trunk fails, later handlers skipped", data: data, trunkErr: errFail, callTrunk: true, wantErr: true},
		{name: "extension fails, domain skipped", data: data, extErr: errFail, callTrunk: true, callExt: true, wantErr: true},
		{name: "customer domain fails", data: data, domainErr: errFail, callTrunk: true, callExt: true, callDomain: true, wantErr: true},
		{name: "all succeed", data: data, callTrunk: true, callExt: true, callDomain: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockExtension := extensionhandler.NewMockExtensionHandler(mc)
			mockTrunk := trunkhandler.NewMockTrunkHandler(mc)
			mockCustomerDomain := customerdomainhandler.NewMockCustomerDomainHandler(mc)

			h := subscribeHandler{
				extensionHandler:      mockExtension,
				trunkHandler:          mockTrunk,
				customerDomainHandler: mockCustomerDomain,
			}

			if tt.callTrunk {
				mockTrunk.EXPECT().EventCUCustomerDeleted(gomock.Any(), cu).Return(tt.trunkErr).Times(1)
			}
			if tt.callExt {
				mockExtension.EXPECT().EventCUCustomerDeleted(gomock.Any(), cu).Return(tt.extErr).Times(1)
			}
			if tt.callDomain {
				mockCustomerDomain.EXPECT().EventCUCustomerDeleted(gomock.Any(), cu).Return(tt.domainErr).Times(1)
			}

			err := h.processEventCMCustomerDeleted(context.Background(), &sock.Event{
				Type: cucustomer.EventTypeCustomerDeleted,
				Data: tt.data,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if (tt.trunkErr != nil || tt.extErr != nil || tt.domainErr != nil) && !errors.Is(err, errFail) {
				t.Errorf("error = %v, want it to wrap the handler error", err)
			}
		})
	}
}

func Test_processEventCMCustomerCreated_errors(t *testing.T) {
	errFail := errors.New("handler failed")

	tests := []struct {
		name string
		data []byte

		domainErr  error
		callDomain bool
	}{
		{name: "invalid json", data: []byte(`{"id":`)},
		{name: "customer domain fails", data: []byte(`{"id":"9a1b0c2e-7f7f-11ee-8f5a-1f2e3d4c5b6a"}`), domainErr: errFail, callDomain: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCustomerDomain := customerdomainhandler.NewMockCustomerDomainHandler(mc)
			h := subscribeHandler{customerDomainHandler: mockCustomerDomain}

			if tt.callDomain {
				mockCustomerDomain.EXPECT().EventCUCustomerCreated(gomock.Any(), gomock.Any()).Return(tt.domainErr).Times(1)
			}

			err := h.processEventCMCustomerCreated(context.Background(), &sock.Event{
				Type: cucustomer.EventTypeCustomerCreated,
				Data: tt.data,
			})
			if err == nil {
				t.Fatalf("expected an error")
			}
			if tt.domainErr != nil && !errors.Is(err, tt.domainErr) {
				t.Errorf("error = %v, want it to wrap the handler error", err)
			}
		})
	}
}
