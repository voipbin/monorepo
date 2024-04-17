package subscribehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmconfbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	commonoutline "gitlab.com/voipbin/bin-manager/common-handler.git/models/outline"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/transcribehandler"
)

func Test_processEvent_processEventCMCallHangup(t *testing.T) {

	tests := []struct {
		name  string
		event *rabbitmqhandler.Event

		expectCall *cmcall.Call
	}{
		{
			name: "normal",

			event: &rabbitmqhandler.Event{
				Publisher: string(commonoutline.ServiceNameCallManager),
				Type:      cmcall.EventTypeCallHangup,
				DataType:  "application/json",
				Data:      []byte(`{"id":"961e297a-f2f1-11ee-a261-57833f5b870a"}`),
			},

			expectCall: &cmcall.Call{
				ID: uuid.FromStringOrNil("961e297a-f2f1-11ee-a261-57833f5b870a"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockTranscribe := transcribehandler.NewMockTranscribeHandler(mc)

			h := subscribeHandler{
				rabbitSock:        mockSock,
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
		event *rabbitmqhandler.Event

		expectConfbridge *cmconfbridge.Confbridge
	}{
		{
			name: "normal",

			event: &rabbitmqhandler.Event{
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

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockTranscribe := transcribehandler.NewMockTranscribeHandler(mc)

			h := subscribeHandler{
				rabbitSock:        mockSock,
				transcribeHandler: mockTranscribe,
			}

			mockTranscribe.EXPECT().EventCMConfbridgeTerminated(gomock.Any(), tt.expectConfbridge).Return(nil)

			if errProcess := h.processEvent(tt.event); errProcess != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errProcess)
			}
		})
	}
}
