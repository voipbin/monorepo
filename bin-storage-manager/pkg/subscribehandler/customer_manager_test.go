package subscribehandler

import (
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-storage-manager/pkg/accounthandler"
	"monorepo/bin-storage-manager/pkg/filehandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
)

func Test_processEvent_processEventCMCustomerCreated(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectCustomer *cmcustomer.Customer
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: "customer-manager",
				Type:      cmcustomer.EventTypeCustomerCreated,
				DataType:  "application/json",
				Data:      []byte(`{"id":"912b0e80-1b6c-11ef-afbd-1f0d484a0e1e"}`),
			},

			expectCustomer: &cmcustomer.Customer{
				ID: uuid.FromStringOrNil("912b0e80-1b6c-11ef-afbd-1f0d484a0e1e"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockFile := filehandler.NewMockFileHandler(mc)

			h := subscribeHandler{
				sockHandler:    mockSock,
				accountHandler: mockAccount,
				fileHandler:    mockFile,
			}

			mockAccount.EXPECT().EventCustomerCreated(gomock.Any(), tt.expectCustomer).Return(nil)

			h.processEvent(tt.event)
		})
	}
}

func Test_processEvent_processEventCMCustomerDeleted(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectCustomer *cmcustomer.Customer
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: "customer-manager",
				Type:      cmcustomer.EventTypeCustomerDeleted,
				DataType:  "application/json",
				Data:      []byte(`{"id":"91a665c6-1b6c-11ef-a253-6768b22f52d4"}`),
			},

			expectCustomer: &cmcustomer.Customer{
				ID: uuid.FromStringOrNil("91a665c6-1b6c-11ef-a253-6768b22f52d4"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockFile := filehandler.NewMockFileHandler(mc)

			h := subscribeHandler{
				sockHandler:    mockSock,
				accountHandler: mockAccount,
				fileHandler:    mockFile,
			}

			mockFile.EXPECT().EventCustomerDeleted(gomock.Any(), tt.expectCustomer).Return(nil)
			mockAccount.EXPECT().EventCustomerDeleted(gomock.Any(), tt.expectCustomer).Return(nil)

			h.processEvent(tt.event)
		})
	}
}
