package subscribehandler

import (
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	mmmessage "monorepo/bin-message-manager/models/message"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-billing-manager/pkg/billinghandler"
)

func Test_processEventMMMessageCreated(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectMessage *mmmessage.Message
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: "message-manager",
				Type:      mmmessage.EventTypeMessageCreated,
				DataType:  "application/json",
				Data:      []byte(`{"id":"bacfd05c-0a03-11ee-aff4-37ba74639f85"}`),
			},

			expectMessage: &mmmessage.Message{
				ID: uuid.FromStringOrNil("bacfd05c-0a03-11ee-aff4-37ba74639f85"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockBilling := billinghandler.NewMockBillingHandler(mc)

			h := subscribeHandler{
				sockHandler:    mockSock,
				billingHandler: mockBilling,
			}

			mockBilling.EXPECT().EventMMMessageCreated(gomock.Any(), tt.expectMessage).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
