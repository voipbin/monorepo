package subscribehandler

import (
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	ememail "monorepo/bin-email-manager/models/email"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/pkg/billinghandler"
)

func Test_processEventEMEmailCreated(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectEmail *ememail.Email
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: "email-manager",
				Type:      ememail.EventTypeCreated,
				DataType:  "application/json",
				Data:      []byte(`{"id":"bacfd05c-0a03-11ee-aff4-37ba74639f85"}`),
			},

			expectEmail: &ememail.Email{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bacfd05c-0a03-11ee-aff4-37ba74639f85"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockBilling := billinghandler.NewMockBillingHandler(mc)

			h := subscribeHandler{
				sockHandler:    mockSock,
				billingHandler: mockBilling,
			}

			mockBilling.EXPECT().EventEMEmailCreated(gomock.Any(), tt.expectEmail).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
