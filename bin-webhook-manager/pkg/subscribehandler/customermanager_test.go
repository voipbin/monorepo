package subscribehandler

import (
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

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
