package subscribehandler

import (
	"errors"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/billinghandler"
	"monorepo/bin-billing-manager/pkg/failedeventhandler"
)

func Test_processEvent_customerFrozenAndRecovered(t *testing.T) {

	customerID := uuid.FromStringOrNil("d5b4a056-0ad4-11ee-b813-bfb4ab48539d")
	errHandler := errors.New("handler failure")

	type test struct {
		name  string
		event *sock.Event

		expectFrozen    bool
		expectRecovered bool
		responseErr     error

		expectErrIs  error
		expectAnyErr bool
	}

	tests := []test{
		{
			name: "frozen",
			event: &sock.Event{
				Publisher: "customer-manager",
				Type:      cucustomer.EventTypeCustomerFrozen,
				DataType:  "application/json",
				Data:      []byte(`{"id":"d5b4a056-0ad4-11ee-b813-bfb4ab48539d"}`),
			},

			expectFrozen: true,
		},
		{
			name: "recovered",
			event: &sock.Event{
				Publisher: "customer-manager",
				Type:      cucustomer.EventTypeCustomerRecovered,
				DataType:  "application/json",
				Data:      []byte(`{"id":"d5b4a056-0ad4-11ee-b813-bfb4ab48539d"}`),
			},

			expectRecovered: true,
		},
		{
			name: "frozen handler error is returned unchanged",
			event: &sock.Event{
				Publisher: "customer-manager",
				Type:      cucustomer.EventTypeCustomerFrozen,
				DataType:  "application/json",
				Data:      []byte(`{"id":"d5b4a056-0ad4-11ee-b813-bfb4ab48539d"}`),
			},

			expectFrozen: true,
			responseErr:  errHandler,
			expectErrIs:  errHandler,
		},
		{
			name: "recovered handler error is returned unchanged",
			event: &sock.Event{
				Publisher: "customer-manager",
				Type:      cucustomer.EventTypeCustomerRecovered,
				DataType:  "application/json",
				Data:      []byte(`{"id":"d5b4a056-0ad4-11ee-b813-bfb4ab48539d"}`),
			},

			expectRecovered: true,
			responseErr:     errHandler,
			expectErrIs:     errHandler,
		},
		{
			name: "frozen with invalid data",
			event: &sock.Event{
				Publisher: "customer-manager",
				Type:      cucustomer.EventTypeCustomerFrozen,
				DataType:  "application/json",
				Data:      []byte(`{"id":`),
			},

			expectAnyErr: true,
		},
		{
			name: "recovered with invalid data",
			event: &sock.Event{
				Publisher: "customer-manager",
				Type:      cucustomer.EventTypeCustomerRecovered,
				DataType:  "application/json",
				Data:      []byte(`not json`),
			},

			expectAnyErr: true,
		},
		{
			name: "frozen from another publisher",
			event: &sock.Event{
				Publisher: "call-manager",
				Type:      cucustomer.EventTypeCustomerFrozen,
				DataType:  "application/json",
				Data:      []byte(`{"id":"d5b4a056-0ad4-11ee-b813-bfb4ab48539d"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := subscribeHandler{
				sockHandler:    sockhandler.NewMockSockHandler(mc),
				accountHandler: mockAccount,
				billingHandler: billinghandler.NewMockBillingHandler(mc),
			}

			expectCustomer := &cucustomer.Customer{ID: customerID}
			if tt.expectFrozen {
				mockAccount.EXPECT().EventCUCustomerFrozen(gomock.Any(), expectCustomer).Return(tt.responseErr).Times(1)
			}
			if tt.expectRecovered {
				mockAccount.EXPECT().EventCUCustomerRecovered(gomock.Any(), expectCustomer).Return(tt.responseErr).Times(1)
			}

			err := h.processEvent(tt.event)
			switch {
			case tt.expectErrIs != nil:
				if err != tt.expectErrIs {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErrIs, err)
				}
			case tt.expectAnyErr:
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
			default:
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}
		})
	}
}

func Test_GetEventProcessor(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockAccount := accounthandler.NewMockAccountHandler(mc)

	var sh SubscribeHandler = &subscribeHandler{
		sockHandler:    sockhandler.NewMockSockHandler(mc),
		accountHandler: mockAccount,
		billingHandler: billinghandler.NewMockBillingHandler(mc),
	}

	processor := GetEventProcessor(sh)

	mockAccount.EXPECT().EventCUCustomerFrozen(gomock.Any(), &cucustomer.Customer{
		ID: uuid.FromStringOrNil("d5b4a056-0ad4-11ee-b813-bfb4ab48539d"),
	}).Return(nil).Times(1)

	if err := processor(&sock.Event{
		Publisher: "customer-manager",
		Type:      cucustomer.EventTypeCustomerFrozen,
		DataType:  "application/json",
		Data:      []byte(`{"id":"d5b4a056-0ad4-11ee-b813-bfb4ab48539d"}`),
	}); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_SetFailedEventHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// the handler that is set at construction must not be used after the replacement.
	mockInitial := failedeventhandler.NewMockFailedEventHandler(mc)
	mockInjected := failedeventhandler.NewMockFailedEventHandler(mc)

	var sh SubscribeHandler = &subscribeHandler{
		sockHandler:        sockhandler.NewMockSockHandler(mc),
		accountHandler:     accounthandler.NewMockAccountHandler(mc),
		billingHandler:     billinghandler.NewMockBillingHandler(mc),
		failedEventHandler: mockInitial,
	}
	SetFailedEventHandler(sh, mockInjected)

	// customer_frozen with invalid data fails in processEvent, so the failed event is saved.
	event := &sock.Event{
		Publisher: "customer-manager",
		Type:      cucustomer.EventTypeCustomerFrozen,
		DataType:  "application/json",
		Data:      []byte(`{"id":`),
	}

	// the error that processEvent returns for this event is deterministic (invalid data),
	// and Save must receive it as it is.
	h := sh.(*subscribeHandler)
	expectErr := h.processEvent(event)
	if expectErr == nil {
		t.Fatalf("Wrong match. expect: processEvent fails for the broken event, got: nil")
	}

	called := false
	mockInjected.EXPECT().Save(gomock.Any(), event, gomock.Any()).DoAndReturn(
		func(_ interface{}, _ *sock.Event, processingErr error) error {
			called = true
			if processingErr == nil || processingErr.Error() != expectErr.Error() {
				t.Errorf("Wrong match. expect error: %v, got: %v", expectErr, processingErr)
			}
			return nil
		},
	).Times(1)

	if err := h.processEventRun(event); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if !called {
		t.Errorf("Wrong match. expect: Save is called on the injected handler, got: no call")
	}
}
