package subscribehandler

import (
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-transcribe-manager/pkg/transcribehandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_processEvent_processEventCMCallHangup(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectCall *cmcall.Call
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCallManager),
				Type:      cmcall.EventTypeCallHangup,
				DataType:  "application/json",
				Data:      []byte(`{"id":"961e297a-f2f1-11ee-a261-57833f5b870a"}`),
			},

			expectCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("961e297a-f2f1-11ee-a261-57833f5b870a"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTranscribe := transcribehandler.NewMockTranscribeHandler(mc)

			h := subscribeHandler{
				sockHandler:       mockSock,
				transcribeHandler: mockTranscribe,
			}

			mockTranscribe.EXPECT().EventCMCallHangup(gomock.Any(), tt.expectCall).Return(nil)

			if errProcess := h.processEvent(tt.event); errProcess != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errProcess)
			}
		})
	}
}

func Test_processEvent_processEventCMConfbridgeTerminated(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectConfbridge *cmconfbridge.Confbridge
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCallManager),
				Type:      cmconfbridge.EventTypeConfbridgeTerminated,
				DataType:  "application/json",
				Data:      []byte(`{"id":"d0f35cb4-f2f1-11ee-891e-376de80e03da"}`),
			},

			expectConfbridge: &cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("d0f35cb4-f2f1-11ee-891e-376de80e03da"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTranscribe := transcribehandler.NewMockTranscribeHandler(mc)

			h := subscribeHandler{
				sockHandler:       mockSock,
				transcribeHandler: mockTranscribe,
			}

			mockTranscribe.EXPECT().EventCMConfbridgeTerminated(gomock.Any(), tt.expectConfbridge).Return(nil)

			if errProcess := h.processEvent(tt.event); errProcess != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errProcess)
			}
		})
	}
}
