package subscribehandler

import (
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	tmspeaking "monorepo/bin-tts-manager/models/speaking"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/pkg/billinghandler"
)

func Test_processEventTTSSpeakingStarted(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectSpeaking *tmspeaking.Speaking
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameTTSManager),
				Type:      tmspeaking.EventTypeSpeakingStarted,
				DataType:  "application/json",
				Data:      []byte(`{"id":"aa111111-0000-0000-0000-000000000001"}`),
			},

			expectSpeaking: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("aa111111-0000-0000-0000-000000000001"),
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

			mockBilling.EXPECT().EventTTSSpeakingStarted(gomock.Any(), tt.expectSpeaking).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEventTTSSpeakingStopped(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectSpeaking *tmspeaking.Speaking
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameTTSManager),
				Type:      tmspeaking.EventTypeSpeakingStopped,
				DataType:  "application/json",
				Data:      []byte(`{"id":"bb111111-0000-0000-0000-000000000001"}`),
			},

			expectSpeaking: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bb111111-0000-0000-0000-000000000001"),
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

			mockBilling.EXPECT().EventTTSSpeakingStopped(gomock.Any(), tt.expectSpeaking).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
