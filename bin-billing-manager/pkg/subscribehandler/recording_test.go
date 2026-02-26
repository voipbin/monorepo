package subscribehandler

import (
	"testing"

	cmrecording "monorepo/bin-call-manager/models/recording"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/pkg/billinghandler"
)

func Test_processEventCMRecordingStarted(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectRecording *cmrecording.Recording
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCallManager),
				Type:      cmrecording.EventTypeRecordingStarted,
				DataType:  "application/json",
				Data:      []byte(`{"id":"cc111111-0000-0000-0000-000000000001"}`),
			},

			expectRecording: &cmrecording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cc111111-0000-0000-0000-000000000001"),
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

			mockBilling.EXPECT().EventCMRecordingStarted(gomock.Any(), tt.expectRecording).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEventCMRecordingFinished(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectRecording *cmrecording.Recording
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCallManager),
				Type:      cmrecording.EventTypeRecordingFinished,
				DataType:  "application/json",
				Data:      []byte(`{"id":"dd111111-0000-0000-0000-000000000001"}`),
			},

			expectRecording: &cmrecording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dd111111-0000-0000-0000-000000000001"),
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

			mockBilling.EXPECT().EventCMRecordingFinished(gomock.Any(), tt.expectRecording).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
